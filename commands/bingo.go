package commands

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/MattSwanson/burtbot/helix"
	"github.com/gempir/go-twitch-irc/v2"
	"github.com/google/btree"
)

const (
	cardCost = 10 // Cost in tokens for a bingo card
	drawTime = 5  // Time between drawing numbers in seconds
	waitTime = 30 // Time to wait before a round begins in seconds
)

var currentGame *Bingo = &Bingo{
	hopper: []ball{},
	players: make(map[string]player),
}
var cardTpl *template.Template

func init() {
	currentGame.drawnNumbers = btree.New(2)
	cardTpl = template.Must(template.ParseFiles("templates/bingo.gohtml"))
	http.HandleFunc("/bingo", DisplayCards)
	RegisterCommand("bingo", currentGame)
}

type Bingo struct {
	hopper           []ball
	drawnNumbers	 *btree.BTree
	players          map[string]player
	running          bool
	drawCancelFunc   context.CancelFunc
}

type player struct {
	user       helix.TwitchUser
	card       []int
	markedCard []int
	url        string
}

type ball int

func (b *Bingo) Run(msg twitch.PrivateMessage) {
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if len(args) == 1 {
		if !b.running {
			return
		}
		if args[0] == "BINGO" {
			if _, ok := b.players[msg.User.DisplayName]; !ok {
				return
			}
			// validate - eventually we want to queue users up
			// so if someone didn't actually have bingo the next
			// person in line gets checked as so on
			if b.validateWinner(msg.User.DisplayName) {
				// winrar
				numTokens := cardCost * len(b.players)
				comm.ToChat(msg.Channel, fmt.Sprintf("@%s has Bingo! They win %d tokens!", msg.User.DisplayName, numTokens))
				comm.ToOverlay(fmt.Sprintf("bingo winner %s %d", msg.User.DisplayName, numTokens))
				// alot tokens to winrar
				GrantToken(msg.User.DisplayName, numTokens)
				b.drawCancelFunc()
				b.running = false
				b.Start(msg.Channel)
				return
			}
			comm.ToChat(msg.Channel, fmt.Sprintf("@%s, it would appear you don't have bingo.", msg.User.DisplayName))
		}
		return
	}
	// !bingo start
	if args[1] == "start" && IsMod(msg.User) {
		b.Start(msg.Channel)
		return
	}

	// !bingo join
	if args[1] == "join" && b.running {
		if _, ok := b.players[msg.User.DisplayName]; ok {
			// user already joined
			comm.ToChat(msg.Channel, fmt.Sprintf("@%s, you've already joined.", msg.User.DisplayName))
			return
		}

		// check to see if they have enough tokens
		if !DeductTokens(msg.User.DisplayName, cardCost) {
			comm.ToChat(msg.Channel, fmt.Sprintf("@%s, bingo cards cost %d tokens. You have only %d.", 
				msg.User.DisplayName, cardCost, GetTokenCount(msg.User)))
			return
		}
		url, err := b.userJoined(msg.User.DisplayName)
		if err != nil {
			GrantToken(msg.User.DisplayName, cardCost)
			comm.ToChat(msg.Channel, fmt.Sprintf("Sorry @%s, couldn't get you resgistered. Try again later. Your tokens have been refunded.", msg.User.DisplayName))
			return
		}
		comm.ToChat(msg.Channel, fmt.Sprintf("@%s see your card here: %s", msg.User.DisplayName, url))
	}

	if args[1] == "stop" && b.running && IsMod(msg.User) {
		comm.ToChat(msg.Channel, "Deactivating Bingo circuits.")
		b.running = false
		b.drawCancelFunc()
		comm.ToOverlay("bingo reset")
	}

}

func NewBingo() *Bingo {
	currentGame.drawnNumbers = btree.New(2)
	return currentGame
}

func (b *Bingo) PostInit() {

}

func (b *Bingo) Start(channelName string) {
		b.running = true
		if b.drawCancelFunc != nil {
			b.drawCancelFunc()
		}
		comm.ToOverlay("bingo reset")
		rand.Seed(time.Now().UnixNano())
		b.players = map[string]player{}
		b.fillHopper()
		b.drawnNumbers.Clear(false)
		b.drawnNumbers.ReplaceOrInsert(btree.Int(0))
		ctx, cancelFunc := context.WithCancel(context.Background())
		b.drawCancelFunc = cancelFunc
		go func(ctx context.Context, chatChannel string) {
			comm.ToChat(chatChannel, fmt.Sprintf("A new round of bingo will start in %d seconds.", waitTime))
			comm.ToChat(chatChannel, fmt.Sprintf("Type !bingo join to buy a card for %d tokens.", cardCost))
			// Wait 30 seconds for people to join before starting
			time.Sleep(time.Second * waitTime)
			if b.running {
				comm.ToChat(chatChannel, "Bingo will now commence.")
			}
			t := time.NewTicker(time.Second * drawTime)
			defer t.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-t.C:
					if len(b.hopper) <= 0 {
						comm.ToChat(chatChannel, "There are no balls left... is anyone even paying attention?")
						comm.ToChat(chatChannel, "Looks like no one won bingo... starting another game soon")
						b.Start(chatChannel)
						return
					}
					drawn := b.drawBall()
					b.drawnNumbers.ReplaceOrInsert(btree.Int(drawn))
					// send a message to the overlay
					// with the drawn number
					letter := ""
					switch {
						case drawn <= 15:
							letter = "B"
							break
						case drawn <= 30:
							letter = "I"
							break
						case drawn <= 45:
							letter = "N"
							break
						case drawn <= 60:
							letter = "G"
							break
						default:
							letter = "O"
					}
					comm.ToOverlay(fmt.Sprintf("bingo drawn %s%d", letter, drawn))
				}
			}
		}(ctx, channelName)

}

func (b Bingo) Help() []string {
	return []string{
		fmt.Sprintf("!bingo join to get a bingo card for %d tokens.", cardCost),
		"Go to the link provided to see your bingo card",
		"!BINGO to call out a bingo",
	}
}

func (b *Bingo) fillHopper() {
	b.hopper = []ball{}
	for i := 1; i <= 75; i++ {
		b.hopper = append(b.hopper, ball(i))
	}
	for i := 0; i < 10; i++ {
		b.shuffleHopper()
	}
}

func (b *Bingo) drawBall() ball {
	// shuffle the hopper
	for i := 0; i < 5; i++ {
		b.shuffleHopper()
	}
	var drawnBall ball
	drawnBall, b.hopper = b.hopper[len(b.hopper)-1], b.hopper[:len(b.hopper)-1]
	return drawnBall
}

func (b *Bingo) shuffleHopper() {
	rand.Shuffle(len(b.hopper), func(i, j int) {
		b.hopper[i], b.hopper[j] = b.hopper[j], b.hopper[i]
	})
}

func (b *Bingo) userJoined(username string) (string, error) {
	// get twitch user info
	user := helix.GetUser(username)
	if user.UserID == "" {
		return "", errors.New("couldn't get user info from twitch")
	}
	// generate a card for them
	card := generateCard()
	// store them in players
	url := fmt.Sprintf("https://burtbot.app/bingo?user=%s", user.DisplayName)
	p := player{user, card, []int{}, url}
	b.players[user.DisplayName] = p
	return url, nil
}

func (b *Bingo) validateWinner(username string) bool {

	if b.drawnNumbers.Has(btree.Int(b.players[username].card[0])) && 
	   b.drawnNumbers.Has(btree.Int(b.players[username].card[6])) && 
	   b.drawnNumbers.Has(btree.Int(b.players[username].card[18])) && 
	   b.drawnNumbers.Has(btree.Int(b.players[username].card[24])) {
		return true
	}
	if b.drawnNumbers.Has(btree.Int(b.players[username].card[4])) && 
	   b.drawnNumbers.Has(btree.Int(b.players[username].card[8])) && 
	   b.drawnNumbers.Has(btree.Int(b.players[username].card[16])) && 
	   b.drawnNumbers.Has(btree.Int(b.players[username].card[20])) {
		return true
	}
	for i := 0; i <= 4; i++ {
		if b.drawnNumbers.Has(btree.Int(b.players[username].card[i])) && 
		   b.drawnNumbers.Has(btree.Int(b.players[username].card[i+5])) && 
		   b.drawnNumbers.Has(btree.Int(b.players[username].card[i+10])) && 
		   b.drawnNumbers.Has(btree.Int(b.players[username].card[i+15])) && 
		   b.drawnNumbers.Has(btree.Int(b.players[username].card[i+20])) {
			return true
		}
	}
	for i := 0; i <= 20; i += 5 {
		if b.drawnNumbers.Has(btree.Int(b.players[username].card[i])) && 
		   b.drawnNumbers.Has(btree.Int(b.players[username].card[i+1])) && 
		   b.drawnNumbers.Has(btree.Int(b.players[username].card[i+2])) && 
		   b.drawnNumbers.Has(btree.Int(b.players[username].card[i+3])) && 
		   b.drawnNumbers.Has(btree.Int(b.players[username].card[i+4])) {
			return true
		}
	}
	return false
}

func DisplayCards(w http.ResponseWriter, r *http.Request) {
	if !currentGame.running {
		http.Error(w, "FORBIDDEN", http.StatusForbidden)
		return
	}
	username := r.FormValue("user")
	p, ok := currentGame.players[username]; 
	if !ok {
		http.Error(w, "I'm a teapot", http.StatusTeapot)
		return
	}
	//card := player.card
	//fmt.Fprintf(w, "user: %s - card: %v\n", username, card)
	// use a template to display the users cards
	d := struct {
		Name   string
		ImgSrc string
		Card   []int
	}{
		Name:   username,
		ImgSrc: p.user.ProfileImgURL,
		Card:   p.card,
	}
	err := cardTpl.ExecuteTemplate(w, "bingo.gohtml", d)
	if err != nil {
		fmt.Fprint(w, err.Error())
	}
}

func generateCard() []int {
	card := make([]int, 25)
	for i := 0; i < 5; i++ {
		colnums := []int{}
		for j := 1; j <= 15; j++ {
			colnums = append(colnums, j+i*15)
		}
		for m := 0; m < 10; m++ {
			rand.Shuffle(len(colnums), func(k, l int) {
				colnums[k], colnums[l] = colnums[l], colnums[k]
			})
		}
		for n := 0; n < 5; n++ {
			card[i+n*5] = colnums[n]
		}
	}
	card[12] = 0
	return card
}
