package commands

import (
	"encoding/json"
	"log"
	"os"

	"github.com/gempir/go-twitch-irc/v2"
)

type BurtCoin struct {
	Wallets map[string]int
}

func (bc *BurtCoin) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	client.Say(msg.Channel, "burtcoin does not exist yet. How will you buy tokens?!!?")
}

// Give
// Take
func (bc BurtCoin) Deduct(user twitch.User, amount int) bool {
	log.Println(bc.Wallets)
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
