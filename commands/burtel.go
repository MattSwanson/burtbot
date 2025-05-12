package commands

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/MattSwanson/burtbot/helix"
	"github.com/gempir/go-twitch-irc/v2"
)

const (
	MinScore        = 500 // Minimum score required to score at start of game
	ScoreFourOf     = 1000
	ScoreFiveOf     = 2000
	ScoreSixOf      = 3000
	ScoreStraight   = 1500
	ScoreThreePair  = 750
	ScoreTwoTriples = 2500
	ScoreThreshold  = 10000 // Score for a player to pass before starting final round
)

type Burtel struct {
	waitingForPlayers  bool
	gameInProgress     bool
	rollInProgress     bool
	connectedToOverlay bool

	chatChannel string

	// string key is a twitch UserID
	players             map[string]*BurtelPlayer
	turnOrder           []string
	currentTurn         int // index for turn order
	playerOverThreshold string
	currentRoundScore   int
	lastRoll            []int
	diceRemaining       int
	roundNumber         int

	rng *rand.Rand
}

type BurtelPlayer struct {
	helix.TwitchUser
	score int
}

var burtel *Burtel = &Burtel{
	players: make(map[string]*BurtelPlayer),
}

func init() {
	RegisterCommand("burtel", burtel)
	comm.SubscribeToReply("burtel", handleOverlayReply)
}

func (b *Burtel) PostInit() {
	// Doing init steps which require the engine to be ready
}

func (b *Burtel) Run(msg twitch.PrivateMessage) {
	args := strings.Fields(strings.ToLower(msg.Message))
	if len(args) < 2 {
		return
	}
	switch args[1] {
	case "start": // Start the burtel system
		if !helix.GetAuthStatus() {
			comm.ToChat(msg.Channel, "Sorry, dumb streamer needs to auth with Twitch first...")
			return
		}
		if b.waitingForPlayers || b.gameInProgress {
			return
		}
		b.startGame(msg.Channel)
		comm.ToChat(msg.Channel, "A game of Burtel has started! Type !join to ... join.")
		return
	case "join": // A player wishes to join the game
		if !b.waitingForPlayers {
			return
		}
		if !b.addPlayer(msg.User) {
			comm.ToChat(msg.Channel, fmt.Sprintf("@%s, you have already joined the game.", msg.User.DisplayName))
		}
		comm.ToChat(msg.Channel, fmt.Sprintf("@%s has joined the game of Burtel!", msg.User.DisplayName))
		return
	case "begin": // Begin the game with the current players
		if !IsMod(msg.User) || b.gameInProgress {
			return
		}
		if len(b.players) == 0 {
			comm.ToChat(msg.Channel, "Need at least one player to start a game of Burtel.")
			return
		}
		b.beginGame()
		return
	case "keep": // Selecting which dice to keep for scoring
		if !b.rollInProgress {
			return
		}
		if msg.User.ID != b.turnOrder[b.currentTurn] {
			return
		}
		if len(args) < 3 {
			m := fmt.Sprintf("@%s You must select at least one die to keep.", msg.User.DisplayName)
			comm.ToChat(msg.Channel, m)
			return
		}

		if len(args[2:]) > b.diceRemaining {
			comm.ToChat(msg.Channel, fmt.Sprintf("@%s You can not keep more than amount you rolled.", msg.User.DisplayName))
		}
		kept := []int{}
		for i, toKeep := range args[2:] {
			// ignore any extra args we got
			if i > 5 {
				break
			}
			n, err := strconv.Atoi(toKeep)
			if err != nil || n < 1 || n > len(b.lastRoll) {
				comm.ToChat(msg.Channel, "Invalid selection for keeping. Try again")
				return
			}
			kept = append(kept, b.lastRoll[n-1])
		}
		b.diceRemaining -= len(kept)

		if b.diceRemaining == 0 {
			b.diceRemaining = 6
		}

		// score the kept dice
		score := b.scoreDice(kept)
		b.currentRoundScore += score
		comm.ToChat(msg.Channel, fmt.Sprintf("@%s scored %d points!", b.currentPlayerName(), score))
		comm.ToChat(msg.Channel, fmt.Sprintf("You have %d points this round. Roll to continue or Hold to stop", b.currentRoundScore))
		b.rollInProgress = false
	case "hold": // Take your current round score and end your turn
		if msg.User.ID != b.turnOrder[b.currentTurn] {
			return
		}
		plrId := b.turnOrder[b.currentTurn]
		if b.players[plrId].score == 0 && b.currentRoundScore < MinScore {
			comm.ToChatf(b.chatChannel, "@%s, You need at least %d points to start scoring. Keep rolling!",
				b.currentPlayerName(), MinScore)
			return
		}
		b.players[plrId].score += b.currentRoundScore
		comm.ToChatf(msg.Channel, "@%s scores %d points and has %d total points!", b.currentPlayerName(), b.currentRoundScore, b.players[plrId].score)
		if b.players[plrId].score >= ScoreThreshold && b.playerOverThreshold == "" {
			b.playerOverThreshold = plrId
			comm.ToChatf(msg.Channel, "@%s has over %d points! Everyone gets one last turn then highest score wins!",
				b.currentPlayerName(), ScoreThreshold)
		}
		if b.nextPlayer() {
			b.endGame(msg.Channel)
			return
		}
		comm.ToChatf(msg.Channel, "It is now @%s's turn!", b.currentPlayerName())
	case "roll":
		if b.rollInProgress {
			return
		}
		if msg.User.ID != b.turnOrder[b.currentTurn] {
			return
		}
		b.rollInProgress = true
		b.lastRoll = b.rollDice(b.diceRemaining)
		res := createResultString(b.lastRoll, msg.User.DisplayName)
		comm.ToChat(msg.Channel, res)
		// determine if the player has any keepable dice
		scoreable := false
		freq := make(map[int]int)
		for _, v := range b.lastRoll {
			freq[v]++
			if v == 1 || v == 5 {
				scoreable = true
			}
			if freq[v] >= 3 {
				scoreable = true
			}
		}
		if !scoreable {
			comm.ToChat(msg.Channel, fmt.Sprintf("@%s has far.. Burteled! What an embarrassment.", b.currentPlayerName()))
			if b.nextPlayer() {
				b.endGame(msg.Channel)
				return
			}
			comm.ToChat(msg.Channel, fmt.Sprintf("It is now @%s's turn! !burtel roll to roll!", b.currentPlayerName()))
			return
		}
		return
	case "scoring": // Show link to scoring page
		comm.ToChat(msg.Channel, "Burtel scoring page coming soon.")
		//TODO:
		// Page should show rules/how to play/scoring
		// Also show current game scores/turn order/etc..
		return
	case "stop":
		return
	default:
		return
	}
}

// startGame performs init for the game state
func (b *Burtel) startGame(chatChannel string) {
	b.waitingForPlayers = true
	b.players = make(map[string]*BurtelPlayer)
	b.diceRemaining = 6
	b.playerOverThreshold = ""
	b.chatChannel = chatChannel
	b.rng = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func (b *Burtel) currentPlayerName() string {
	id := b.turnOrder[b.currentTurn]
	player, ok := b.players[id]
	if !ok {
		return ""
	}
	return player.DisplayName
}

func (b *Burtel) addPlayer(player twitch.User) bool {
	if _, ok := b.players[player.ID]; ok {
		return false
	}
	newPlayer := BurtelPlayer{
		helix.GetUser(player.Name),
		0,
	}
	b.players[player.ID] = &newPlayer
	b.turnOrder = append(b.turnOrder, player.ID)
	return true
}

func (b *Burtel) beginGame() {
	b.waitingForPlayers = false
	// select a random player to start
	rand.Shuffle(len(b.turnOrder), func(i, j int) {
		b.turnOrder[i], b.turnOrder[j] = b.turnOrder[j], b.turnOrder[i]
	})
	b.currentTurn = 0
	b.roundNumber = 1
	comm.ToChatf(b.chatChannel, "Burtel has begun! @%s goes first! !burtel roll to roll!", b.currentPlayerName())
}

// rollDice gets n random numbers [1:6]
func (b *Burtel) rollDice(n int) []int {
	rolls := []int{}
	for i := 0; i < n; i++ {
		rolls = append(rolls, b.rng.Intn(6)+1)
	}
	return rolls
}

//TODO: support printing what scored, you know
func (b *Burtel) scoreDice(dice []int) int {
	freq := make(map[int]int)
	hasStraight := len(dice) == 6
	for _, v := range dice {
		freq[v]++
		if freq[v] > 1 {
			hasStraight = false
		}
	}

	pairs := []int{}
	trips := []int{}

	score := 0
	//TODO: 5 of 6 of
	for n, f := range freq {
		if f == 6 {
			return ScoreSixOf
		}
		if f == 5 {
			score += ScoreFiveOf
		}
		if f == 4 {
			score += ScoreFourOf
		}
		if f == 3 {
			trips = append(trips, n)
			tripScore := n * 100
			if n == 1 {
				tripScore = 300
			}
			score += tripScore
			continue
		}
		if f == 2 {
			pairs = append(pairs, n)
		}
		if n == 1 {
			score += f * 100
		}
		if n == 5 {
			score += f * 50
		}
	}

	if hasStraight {
		return ScoreStraight
	}
	if len(pairs) == 3 {
		return ScoreThreePair
	}
	if len(trips) == 2 {
		return ScoreTwoTriples
	}

	return score
}

func (b *Burtel) endGame(channel string) {
	winner := b.turnOrder[0]
	for plrId, player := range b.players {
		if player.score > b.players[winner].score {
			winner = plrId
		}
	}
	comm.ToChatf(channel, "Congrats @%s! You won with a score of %d!",
		b.players[winner].DisplayName, b.players[winner].score)
}

// nextPlayer sets up for the next players turn and then returns false.
// Will return true if the game is over.
func (b *Burtel) nextPlayer() bool {
	b.currentRoundScore = 0
	b.currentTurn = (b.currentTurn + 1) % len(b.turnOrder)
	b.lastRoll = []int{}
	b.diceRemaining = 6
	if b.turnOrder[b.currentTurn] == b.playerOverThreshold {
		return true
	}
	if b.currentTurn == 0 {
		b.roundNumber++
	}
	b.rollInProgress = false
	return false
}

func createResultString(dice []int, userName string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "@%s rolled:", userName)
	for _, value := range dice {
		fmt.Fprintf(&b, " %d", value)
	}
	return b.String()
}

func handleOverlayReply(args []string) {
	// If we get a roll over message
	// Unlock the game

}

func (b *Burtel) Help() []string {
	return []string{
		"start - Start up a game of Burtel",
		"join - To join the game",
		"begin - Begin a game with the current joined players",
		"keep [1|2|3|4|5|6] - Keep all of the given dice for scoring",
		"hold - End your turn",
		"roll - Roll the available dice",
		"scoring - Get a link to page with scoring/rules",
	}
}
