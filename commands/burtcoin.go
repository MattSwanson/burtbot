package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/gempir/go-twitch-irc/v2"
)

type BurtCoin struct {
	Wallets map[string]int
}

func (bc *BurtCoin) Init() {
	// burtcoin init
	bc.Wallets = make(map[string]int)
	j, err := os.ReadFile("./wallets.json")
	if err != nil {
		log.Println("Couldn't load burtcoin wallet info from file")
	} else {
		err = json.Unmarshal(j, &bc.Wallets)
		if err != nil {
			log.Println("Invalid json in tokens file")
		}
	}
	//burtCoin := &commands.BurtCoin{Wallets: wallets}
}

func (bc *BurtCoin) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if args[1] == "give" {
		if len(args) < 4 {
			return
		}

		n, err := strconv.Atoi(args[3])
		if err != nil {
			return
		}
		if bc.Give(msg.User, args[2], n) {
			client.Say(msg.Channel, fmt.Sprintf("@%s gave %d burtcoin to %s. How nice.", msg.User.DisplayName, n, args[2]))
		} else {
			client.Say(msg.Channel, fmt.Sprintf("@%s you don't have enough burtcoin to give %d.", msg.User.DisplayName, n))
		}
	}

	if args[1] == "balance" {
		client.Say(msg.Channel, fmt.Sprintf("@%s you have %d burtcoin.", msg.User.DisplayName, bc.Balance(msg.User)))
		return
	}
}

// Give
func (bc BurtCoin) Give(giver twitch.User, recipient string, amount int) bool {
	if !bc.Deduct(giver, amount) {
		return false
	}
	bc.Wallets[recipient] += amount
	bc.saveWalletsToFile()
	return true
}

// Take
func (bc BurtCoin) Deduct(user twitch.User, amount int) bool {
	if amount > bc.Wallets[user.Name] {
		return false
	}
	bc.Wallets[user.Name] -= amount
	bc.saveWalletsToFile()
	return true
}

// Mine
// user starts the "miner"
// and then when they leave the channel it stops
// OnUserPart is callback for when a user "parts" the channel

// Balance
func (bc BurtCoin) Balance(user twitch.User) int {
	return bc.Wallets[user.Name]
}

func (bc BurtCoin) saveWalletsToFile() {
	json, err := json.Marshal(bc.Wallets)
	if err != nil {
		log.Println("Couldn't json")
		return
	}
	if err := os.WriteFile("./wallets.json", json, 0644); err != nil {
		log.Println(err.Error())
	}
}
