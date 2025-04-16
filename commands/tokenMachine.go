package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/MattSwanson/burtbot/helix"
	"github.com/gempir/go-twitch-irc/v2"
)

const followRewardAmount = 100

var tm *TokenMachine = &TokenMachine{Tokens: make(map[string]*big.Int)}

type TokenMachine struct {
	lastKick            time.Time
	lastDistract        time.Time
	attendantDistracted bool
	Tokens              map[string]*big.Int
	persist             bool
}

const (
	kickCooldown           = 120    // seconds
	kickProbability        = 0.25   // %
	kickJackpotProbability = 0.0001 // %
	jackpotAmount          = 25
	tokenRate              = 4  // tokens/burtcoin
	distractTime           = 30 // seconds
	distractCooldown       = 24 // hours
)

func init() {
	rand.Seed(time.Now().Unix())
	// tokens init
	tm.Tokens = make(map[string]*big.Int)
	tm.persist = true
	j, err := os.ReadFile("./tokens.json")
	if err != nil {
		log.Println("Couldn't load token info from file")
		tm.persist = false
	} else {
		err = json.Unmarshal(j, &tm.Tokens)
		if err != nil {
			log.Println("Invalid json in tokens file")
			tm.persist = false
		}
	}
	RegisterCommand("tokenmachine", tm)
}

func GetTokenMachine() *TokenMachine {
	return tm
}

func (t *TokenMachine) PostInit() {
	helix.SubscribeToFollowEvent(t.FollowReward)
}

func (t *TokenMachine) Run(msg twitch.PrivateMessage) {

	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))

	if len(args) < 2 {
		return
	}

	switch args[1] {
	case "kick":
		t.kick(&msg)
	case "distract":
		t.distract(&msg)
	case "buy":
		if len(args) < 3 {
			return
		}
		r := struct {
			Amount int
		}{}
		if result, err := CheckArgs(args[2:], 1, &r); err != nil || !result {
			return
		}
		t.buyTokens(r.Amount, &msg)
	case "balance":
		t.checkBalance(&msg)
	case "set":
		if !IsMod(msg.User) || len(args) < 4 {
			return
		}
		n := big.NewInt(0)
		_, err := fmt.Sscan(args[3], n)
		if err != nil {
			comm.ToChat(msg.Channel, "Unable to set token amount. Invalid amount")
			return
		}
		t.setTokenCount(args[2], n)
	case "grant":
		if !IsMod(msg.User) || len(args) < 4 {
			return
		}
		n := big.NewInt(0)
		_, err := fmt.Sscan(args[3], n)
		if err != nil {
			comm.ToChat(msg.Channel, "Unable to grant tokens. Invalid amount")
			return
		}
		t.GrantToken(strings.ToLower(args[2]), n)
		comm.ToChat(msg.Channel, fmt.Sprintf("@%s, you were given %d tokens! Use them to play games.", args[2], n))
	case "give":
		if len(args) < 4 {
			return
		}
		n := big.NewInt(0)
		z := big.NewInt(0)
		_, err := fmt.Sscan(args[3], n)
		if err != nil {
			comm.ToChat(msg.Channel, fmt.Sprintf("@%s, there was an error processing your request. Please try again in a moment.", msg.User.DisplayName))
			return
		}
		if n.Cmp(z) <= 0 {
			comm.ToChat(msg.Channel, "You can not give zero or negative amounts of tokens")
			return
		}
		if tokenCount := t.getTokenCount(msg.User.DisplayName); n.Cmp(tokenCount) <= 0 {
			comm.ToChat(msg.Channel, fmt.Sprintf("@%s, you can't give that many tokens, you only have %d.", msg.User.DisplayName, tokenCount))
			return
		}
		DeductTokens(msg.User.DisplayName, n)
		GrantToken(strings.ToLower(args[2]), n)
		comm.ToChat(msg.Channel, fmt.Sprintf("@%s gave %d tokens to @%s! How nice!", msg.User.DisplayName, n, args[2]))
	}

}

func (t *TokenMachine) DeductTokens(username string, number *big.Int) bool {
	balance, _ := t.Tokens[strings.ToLower(username)]
	if balance.Cmp(number) == -1 {
		return false
	}
	t.Tokens[strings.ToLower(username)].Sub(balance, number)
	return true
}

func DeductTokens(username string, number *big.Int) bool {
	return tm.DeductTokens(username, number)
}

func (t *TokenMachine) GrantToken(username string, number *big.Int) {
	cur := t.Tokens[strings.ToLower(username)]
	if cur == nil {
		cur = big.NewInt(0)
	}
	cur.Add(cur, number)
	t.Tokens[strings.ToLower(username)] = cur
	if t.persist {
		t.saveTokensToFile()
	}
}

func GrantToken(username string, number *big.Int) {
	tm.GrantToken(username, number)
}

func (t *TokenMachine) setTokenCount(userName string, number *big.Int) {
	t.Tokens[strings.ToLower(userName)] = number
	if t.persist {
		t.saveTokensToFile()
	}
}

func (t *TokenMachine) saveTokensToFile() {
	json, err := json.Marshal(t.Tokens)
	if err != nil {
		log.Println("Couldn't json")
		return
	}
	if err := os.WriteFile("./tokens.json", json, 0644); err != nil {
		log.Println(err.Error())
	}
}

// Get a user's current token count
func (t *TokenMachine) getTokenCount(username string) *big.Int {
	username = strings.ToLower(username)
	// No one gets any tokens!!!!
	count := t.Tokens[username]
	if count == nil {
		count = big.NewInt(0)
	}
	return count
}

func GetTokenCount(user twitch.User) *big.Int {
	return tm.getTokenCount(user.DisplayName)
}

func (t *TokenMachine) FollowReward(username string) {
	s := fmt.Sprintf("Thanks for following @%s! Have %d tokens to spend on useless things...", username, followRewardAmount)
	comm.ToChat("burtstanton", s)
	t.GrantToken(username, big.NewInt(followRewardAmount))
}

func (t *TokenMachine) buyTokens(amount int, msg *twitch.PrivateMessage) {
	if amount%tokenRate != 0 {
		comm.ToChat(msg.Channel, fmt.Sprintf("@%s, right now tokens are %[2]d for one burtcoin. Please buy in multiples of %[2]d.", msg.User.DisplayName, tokenRate))
		return
	}
	bcBalance := GetBurtcoinBalance(msg.User)
	if bcBalance < 1 {
		comm.ToChat(msg.Channel, fmt.Sprintf("@%s, you don't have any burtcoins with which to buy tokens.", msg.User.Name))
		return
	}
	if float64(amount)/float64(tokenRate) > bcBalance {
		comm.ToChat(msg.Channel, fmt.Sprintf("@%s, you don't have enough burtcoins to buy %d tokens.", msg.User.Name, amount))
		comm.ToChat(msg.Channel, fmt.Sprintf("@%s, you have %.2f. Need %d.", msg.User.Name, bcBalance, amount/tokenRate))
		return
	}

	if DeductBurtcoin(msg.User, float64(amount)/float64(tokenRate)) {
		b := big.NewInt(int64(amount))
		t.GrantToken(strings.ToLower(msg.User.Name), b)
		comm.ToChat(msg.Channel, fmt.Sprintf("@%s, you received %d tokens for %d burtcoin. Thanks!", msg.User.DisplayName, amount, amount/tokenRate))
	} else {
		comm.ToChat(msg.Channel, fmt.Sprintf("@%s, unable to deduct funds from you burtcoin wallet. No tokens for you. Yet...", msg.User.DisplayName))
	}
}

func (t *TokenMachine) checkBalance(msg *twitch.PrivateMessage) {
	n := t.getTokenCount(msg.User.DisplayName)
	if n.Cmp(big.NewInt(0)) == 0 {
		comm.ToChat(msg.Channel, fmt.Sprintf(`@%s, Ya got NONE!`, msg.User.Name))
		return
	}
	plural := ""
	if n.Cmp(big.NewInt(1)) == 1 {
		plural = "s"
	}
	str := fmt.Sprintf(`@%s, you have %d token%s. Use them wisely. Or not.`, msg.User.Name, n, plural)
	if len(str) >= 500 {
		amt := n.String()
		comm.ToChat(msg.Channel, fmt.Sprintf("@%s, you have: ", msg.User.DisplayName))
		for i := 0; i < len(amt); i += 499 {
			if i+499 >= len(amt) {
				comm.ToChat(msg.Channel, amt[i:])
			} else {
				comm.ToChat(msg.Channel, amt[i:i+500])
			}
		}
		comm.ToChat(msg.Channel, "tokens.")
	}
	comm.ToChat(msg.Channel, str)
}

func (t *TokenMachine) distract(msg *twitch.PrivateMessage) {
	if t.getTokenCount(msg.User.DisplayName).Cmp(big.NewInt(0)) == 0 {
		comm.ToChat(msg.Channel, fmt.Sprintf("@%s, you don't have anything to distract the Attendant with.", msg.User.DisplayName))
		return
	}
	if time.Now().Before(t.lastDistract.Add(time.Hour * distractCooldown)) {
		comm.ToChat(msg.Channel, "The Attendant won't fall for those shenanigans again. At least not yet.")
		return
	}
	t.lastDistract = time.Now()
	t.attendantDistracted = true
	t.DeductTokens(msg.User.DisplayName, big.NewInt(1))
	comm.ToChat(msg.Channel, fmt.Sprintf("@%s throws a token into the back hallway.", msg.User.DisplayName))
	comm.ToChat(msg.Channel, "The Attendant goes off to investigate the noise.")
	comm.ToChat(msg.Channel, "Quick! The token machine is unattended, now would be a good check to try and get free tokens!")
	go func() {
		time.Sleep(time.Second * distractTime)
		t.attendantDistracted = false
		comm.ToChat(msg.Channel, "The Attendant returns from checking out the suspicious noise.")
	}()
}

func (t *TokenMachine) kick(msg *twitch.PrivateMessage) {
	if t.attendantDistracted {
		rand.Seed(time.Now().Unix())
		if rand.Float64() >= kickProbability {
			// failed to get a token
			comm.ToChat(msg.Channel, fmt.Sprintf("@%s you bad at kicking machine", msg.User.DisplayName))
			return
		}
		t.GrantToken(strings.ToLower(msg.User.Name), big.NewInt(1))
		comm.ToChat(msg.Channel, fmt.Sprintf("DING! @%s got a free token!", msg.User.DisplayName))
		return
	}

	if time.Now().Before(t.lastKick.Add(time.Second * kickCooldown)) {
		comm.ToChat(msg.Channel, "You can't kick the machine with the Attendant watching. Give it time.")
		return
	}
	t.lastKick = time.Now()

	r := rand.Float64()
	if r >= kickProbability {
		// failed to get a token
		comm.ToChat(msg.Channel, "You kick the token machine but nothing happens.")
		comm.ToChat(msg.Channel, "The Attendant comes to investigate the noise.")
		comm.ToChat(msg.Channel, "You leave the room before he notices you.")
		return
	}

	if r < kickJackpotProbability {
		t.GrantToken(strings.ToLower(msg.User.Name), big.NewInt(jackpotAmount))
		comm.ToChat(msg.Channel, fmt.Sprintf("WOW! @%s kicks the token machine and %d tokens fall from it's orifices.", msg.User.DisplayName, jackpotAmount))
		comm.ToChat(msg.Channel, "They grab their bounty from the floor quickly and get away before The Attendent rushes over.")
		return
	}

	t.GrantToken(strings.ToLower(msg.User.Name), big.NewInt(1))
	comm.ToChat(msg.Channel, "You kick the token machine and a token falls into the tray.")
	comm.ToChat(msg.Channel, "As you grab the token you notice the Attendant coming.")
	comm.ToChat(msg.Channel, "You escape into the shadows with your request token.")
}

func (t *TokenMachine) Help() []string {
	return []string{
		"!tokenmachine buy [amount] to buy tokens with hard earned burtcoin",
		fmt.Sprintf("Get %d tokens for one burtcoin", tokenRate),
		"!tokenmachine balance to see how many tokens you have. For now.",
		"!tokenmachine give [user] [amount] to give that user that amount of tokens out your own pocket.",
	}
}
