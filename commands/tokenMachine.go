package commands

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/gempir/go-twitch-irc/v2"
)

type TokenMachine struct {
	lastKick            time.Time
	lastDistract        time.Time
	attendantDistracted bool
	Music               *Music
	BurtCoin            *BurtCoin
}

const (
	kickCooldown     = 120 // seconds
	kickProbability  = 25  // %
	tokenRate        = 4   // tokens/burtcoin
	distractTime     = 30  // seconds
	distractCooldown = 24  // hours
)

func (t *TokenMachine) Run(client *twitch.Client, msg twitch.PrivateMessage) {

	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))

	if args[1] == "kick" {

		if t.attendantDistracted {
			rand.Seed(time.Now().Unix())
			if rand.Int()%100 >= kickProbability {
				// failed to get a token
				client.Say(msg.Channel, fmt.Sprintf("@%s you bad at kicking machine", msg.User.DisplayName))
				return
			}
			t.Music.grantToken(strings.ToLower(msg.User.Name), 1)
			client.Say(msg.Channel, fmt.Sprintf("DING! @%s got a free token!", msg.User.DisplayName))
			return
		}

		if time.Now().Before(t.lastKick.Add(time.Second * kickCooldown)) {
			client.Say(msg.Channel, "You can't kick the machine with the Attendant watching. Give it time.")
			return
		}
		t.lastKick = time.Now()
		rand.Seed(time.Now().Unix())
		if rand.Int()%100 >= kickProbability {
			// failed to get a token
			client.Say(msg.Channel, "You kick the token machine but nothing happens.")
			client.Say(msg.Channel, "The Attendant comes to investigate the noise.")
			client.Say(msg.Channel, "You leave the room before he notices you.")
			return
		}

		t.Music.grantToken(strings.ToLower(msg.User.Name), 1)
		client.Say(msg.Channel, "You kick the token machine and token falls into the tray.")
		client.Say(msg.Channel, "As you grab the token you notice the Attendant coming.")
		client.Say(msg.Channel, "You escape into the shadows with your request token.")
		return
	}

	if args[1] == "distract" {
		if t.Music.getTokenCount(msg.User) == 0 {
			client.Say(msg.Channel, fmt.Sprintf("@%s, you don't have anything to distract the Attendant with.", msg.User.DisplayName))
			return
		}
		if time.Now().Before(t.lastDistract.Add(time.Hour * distractCooldown)) {
			client.Say(msg.Channel, "The Attendant won't fall for those shenanigans again. At least not yet.")
			return
		}
		t.lastDistract = time.Now()
		t.attendantDistracted = true
		t.Music.setTokenCount(msg.User, t.Music.getTokenCount(msg.User)-1)
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
		if amount/tokenRate > bcBalance {
			client.Say(msg.Channel, fmt.Sprintf("@%s, you don't have enough burtcoins to buy %d tokens.", msg.User.Name, amount))
			client.Say(msg.Channel, fmt.Sprintf("@%s, you have %d. Need %d.", msg.User.Name, bcBalance, amount/tokenRate))
			return
		}

		if t.BurtCoin.Deduct(msg.User, amount/tokenRate) {
			t.Music.grantToken(strings.ToLower(msg.User.Name), amount)
			client.Say(msg.Channel, fmt.Sprintf("@%s, you received %d tokens for %d burtcoin. Thanks!", msg.User.DisplayName, amount, amount/tokenRate))
		} else {
			client.Say(msg.Channel, fmt.Sprintf("@%s, unable to deduct funds from you burtcoin wallet. No tokens for you. Yet...", msg.User.DisplayName))
		}

		return
	}

	if args[1] == "balance" {
		n := t.Music.getTokenCount(msg.User)
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
}
