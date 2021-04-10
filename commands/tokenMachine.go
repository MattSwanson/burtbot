package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gempir/go-twitch-irc/v2"
)

type TokenMachine struct {
	lastKick            time.Time
	lastDistract        time.Time
	attendantDistracted bool
	BurtCoin            *BurtCoin
	Tokens              map[string]int
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

func (tm *TokenMachine) Init() {
	rand.Seed(time.Now().Unix())
	// tokens init
	tm.Tokens = make(map[string]int)
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
}

func (t *TokenMachine) Run(client *twitch.Client, msg twitch.PrivateMessage) {

	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))

	if len(args) < 2 {
		return
	}

	if args[1] == "kick" {

		if t.attendantDistracted {
			rand.Seed(time.Now().Unix())
			if rand.Float64() >= kickProbability {
				// failed to get a token
				client.Say(msg.Channel, fmt.Sprintf("@%s you bad at kicking machine", msg.User.DisplayName))
				return
			}
			t.GrantToken(strings.ToLower(msg.User.Name), 1)
			client.Say(msg.Channel, fmt.Sprintf("DING! @%s got a free token!", msg.User.DisplayName))
			return
		}

		if time.Now().Before(t.lastKick.Add(time.Second * kickCooldown)) {
			client.Say(msg.Channel, "You can't kick the machine with the Attendant watching. Give it time.")
			return
		}
		t.lastKick = time.Now()

		r := rand.Float64()
		if r >= kickProbability {
			// failed to get a token
			client.Say(msg.Channel, "You kick the token machine but nothing happens.")
			client.Say(msg.Channel, "The Attendant comes to investigate the noise.")
			client.Say(msg.Channel, "You leave the room before he notices you.")
			return
		}

		if r < kickJackpotProbability {
			t.GrantToken(strings.ToLower(msg.User.Name), jackpotAmount)
			client.Say(msg.Channel, fmt.Sprintf("WOW! @%s kicks the token machine and %d tokens fall from it's orifices.", msg.User.DisplayName, jackpotAmount))
			client.Say(msg.Channel, "They grab their bounty from the floor quickly and get away before The Attendent rushes over.")
			return
		}

		t.GrantToken(strings.ToLower(msg.User.Name), 1)
		client.Say(msg.Channel, "You kick the token machine and a token falls into the tray.")
		client.Say(msg.Channel, "As you grab the token you notice the Attendant coming.")
		client.Say(msg.Channel, "You escape into the shadows with your request token.")
		return
	}

	if args[1] == "distract" {
		if t.getTokenCount(msg.User) == 0 {
			client.Say(msg.Channel, fmt.Sprintf("@%s, you don't have anything to distract the Attendant with.", msg.User.DisplayName))
			return
		}
		if time.Now().Before(t.lastDistract.Add(time.Hour * distractCooldown)) {
			client.Say(msg.Channel, "The Attendant won't fall for those shenanigans again. At least not yet.")
			return
		}
		t.lastDistract = time.Now()
		t.attendantDistracted = true
		t.setTokenCount(msg.User.DisplayName, t.getTokenCount(msg.User)-1)
		client.Say(msg.Channel, fmt.Sprintf("@%s throws a token into the back hallway.", msg.User.DisplayName))
		client.Say(msg.Channel, "The Attendant goes off to investigate the noise.")
		client.Say(msg.Channel, "Quick! The token machine is unattended, now would be a good check to try and get free tokens!")
		go func() {
			time.Sleep(time.Second * distractTime)
			t.attendantDistracted = false
			client.Say(msg.Channel, "The Attendant returns from checking out the suspicious noise.")
		}()
		return
	}

	if args[1] == "buy" {
		//TODO - not functional yet - also no one has burtcoins yet so not a lie
		if len(args) < 3 {
			return
		}
		amount, err := strconv.Atoi(args[2])
		if err != nil {
			return
		}
		if amount%tokenRate != 0 {
			client.Say(msg.Channel, fmt.Sprintf("@%s, right now tokens are %[2]d for one burtcoin. Please buy in multiples of %[2]d.", msg.User.DisplayName, tokenRate))
			return
		}
		bcBalance := t.BurtCoin.Balance(msg.User)
		if bcBalance < 1 {
			client.Say(msg.Channel, fmt.Sprintf("@%s, you don't have any burtcoins with which to buy tokens.", msg.User.Name))
			return
		}
		if float64(amount)/float64(tokenRate) > bcBalance {
			client.Say(msg.Channel, fmt.Sprintf("@%s, you don't have enough burtcoins to buy %d tokens.", msg.User.Name, amount))
			client.Say(msg.Channel, fmt.Sprintf("@%s, you have %.2f. Need %d.", msg.User.Name, bcBalance, amount/tokenRate))
			return
		}

		if t.BurtCoin.Deduct(msg.User, float64(amount)/float64(tokenRate)) {
			t.GrantToken(strings.ToLower(msg.User.Name), amount)
			client.Say(msg.Channel, fmt.Sprintf("@%s, you received %d tokens for %d burtcoin. Thanks!", msg.User.DisplayName, amount, amount/tokenRate))
		} else {
			client.Say(msg.Channel, fmt.Sprintf("@%s, unable to deduct funds from you burtcoin wallet. No tokens for you. Yet...", msg.User.DisplayName))
		}

		return
	}

	if args[1] == "balance" {
		n := t.getTokenCount(msg.User)
		if n == 0 {
			client.Say(msg.Channel, fmt.Sprintf(`@%s, Ya got NONE!`, msg.User.Name))
			return
		}
		plural := ""
		if n > 1 {
			plural = "s"
		}
		client.Say(msg.Channel, fmt.Sprintf(`@%s, you have %d token%s. Use them wisely. Or not.`, msg.User.Name, n, plural))
	}

	if args[1] == "set" {
		if !isMod(msg.User) || len(args) < 4 {
			return
		}
		n, err := strconv.Atoi(args[3])
		if err != nil {
			log.Println("error converting to int - ", err.Error())
			return
		}
		t.setTokenCount(args[2], n)
	}

	if args[1] == "grant" {
		if !isMod(msg.User) || len(args) < 4 {
			return
		}
		n, err := strconv.Atoi(args[3])
		if err != nil {
			log.Println("error converting to int - ", err.Error())
			return
		}
		t.GrantToken(strings.ToLower(args[2]), n)
		client.Say(msg.Channel, fmt.Sprintf("@%s, you were given %d tokens! Use them to play games.", args[2], n))
	}
}

func (t *TokenMachine) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {
	return
}

func (t *TokenMachine) GrantToken(username string, number int) {
	t.Tokens[username] += number
	if t.persist {
		t.saveTokensToFile()
	}
}

func (t *TokenMachine) setTokenCount(userName string, number int) {
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
func (t *TokenMachine) getTokenCount(user twitch.User) int {
	username := strings.ToLower(user.Name)
	// No one gets any tokens!!!!
	return t.Tokens[username]
}
