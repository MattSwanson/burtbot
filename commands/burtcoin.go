package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

const (
	miningAmount = .005 // per tick
)

var burtCoin *BurtCoin = &BurtCoin{}

type BurtCoin struct {
	Wallets         map[string]float64
	Mining          map[string]context.CancelFunc
	saveTimerCancel context.CancelFunc

	lock sync.Mutex
}

func init() {
	RegisterCommand("burtcoin", burtCoin)
}

func (bc *BurtCoin) PostInit() {
	// burtcoin init
	bc.Wallets = make(map[string]float64)
	bc.Mining = make(map[string]context.CancelFunc)
	j, err := os.ReadFile("./wallets.json")
	if err != nil {
		log.Println("Couldn't load burtcoin wallet info from file")
	} else {
		err = json.Unmarshal(j, &bc.Wallets)
		if err != nil {
			log.Println("Invalid json in tokens file")
		}
	}
	burtCoin = bc
	// spin up a ticker to save the wallets to a file every 5 seconds
	ctx, cancel := context.WithCancel(context.Background())
	bc.saveTimerCancel = cancel
	go bc.startSaveTimer(ctx)
	SubscribeUserPart(bc.OnUserPart)
	SubscribeUserJoin(bc.OnUserJoin)
}

func (bc *BurtCoin) startSaveTimer(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			bc.saveWalletsToFile()
		}
	}
}

func (bc *BurtCoin) Run(msg twitch.PrivateMessage) {
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if len(args) < 2 {
		return
	}
	if args[1] == "give" {
		if len(args) < 4 {
			return
		}

		n, err := strconv.Atoi(args[3])
		if err != nil {
			return
		}
		if n < 0 {
			comm.ToChat(msg.Channel, "Can't give negative amounts of burtcoin")
			return
		}
		if bc.Give(msg.User, args[2], float64(n)) {
			comm.ToChat(msg.Channel, fmt.Sprintf("@%s gave %d burtcoin to %s. How nice.", msg.User.DisplayName, n, args[2]))
		} else {
			comm.ToChat(msg.Channel, fmt.Sprintf("@%s you don't have enough burtcoin to give %d.", msg.User.DisplayName, n))
		}
	}

	if args[1] == "balance" {
		comm.ToChat(msg.Channel, fmt.Sprintf("@%s you have %f burtcoin.", msg.User.DisplayName, bc.Balance(msg.User)))
		return
	}

	if args[1] == "mine" {
		if len(args) < 3 {
			return
		}
		if args[2] == "start" {
			if bc.Mine(msg.User.Name) {
				comm.ToChat(msg.Channel, fmt.Sprintf("@%s has started mining burtcoin. What a waste.", msg.User.DisplayName))
			} else {
				comm.ToChat(msg.Channel, fmt.Sprintf("@%s, you can't start another miner.", msg.User.DisplayName))
			}
			return
		}
		if args[2] == "stop" {
			if bc.StopMining(msg.User.Name) {
				comm.ToChat(msg.Channel, fmt.Sprintf("@%s has stopped mining burtcoin.", msg.User.DisplayName))
			}
			return
		}
	}

}

func (bc *BurtCoin) OnUserPart(msg twitch.UserPartMessage) {
	// log.Println(fmt.Sprintf(`%s has left the channel, close down their miner if app.`, msg.User))
	if bc.StopMining(msg.User) {
		// comm.ToChat(msg.Channel, fmt.Sprintf("%s left - turning off their miner to save my energies... or something.", msg.User))
	}
}

func (bc *BurtCoin) OnUserJoin(msg twitch.UserJoinMessage) {
	// start a miner for the user joining - silently
	bc.Mine(msg.User)
}

// Give
func (bc *BurtCoin) Give(giver twitch.User, recipient string, amount float64) bool {
	if !bc.Deduct(giver, amount) {
		return false
	}
	bc.lock.Lock()
	bc.Wallets[recipient] += amount
	bc.lock.Unlock()
	return true
}

// Take
func (bc *BurtCoin) Deduct(user twitch.User, amount float64) bool {
	bc.lock.Lock()
	if amount > bc.Wallets[user.Name] {
		return false
	}
	bc.Wallets[user.Name] -= amount
	bc.lock.Unlock()
	return true
}

func DeductBurtcoin(user twitch.User, amount float64) bool {
	return burtCoin.Deduct(user, amount)
}

// Mine
func (bc *BurtCoin) Mine(username string) bool {
	if _, ok := bc.Mining[username]; ok {
		return false
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func(ctx context.Context) {
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				bc.lock.Lock()
				bc.Wallets[username] += miningAmount
				bc.lock.Unlock()
			}
		}
	}(ctx)
	bc.Mining[username] = cancel
	return true
}

func (bc *BurtCoin) StopMining(username string) bool {
	bc.lock.Lock()
	defer bc.lock.Unlock()
	if cancelFunc, ok := bc.Mining[username]; ok {
		cancelFunc()
		delete(bc.Mining, username)
		return true
	}
	return false
}

// Balance
func (bc *BurtCoin) Balance(user twitch.User) float64 {
	bc.lock.Lock()
	defer bc.lock.Unlock()
	return bc.Wallets[user.Name]
}

func GetBurtcoinBalance(user twitch.User) float64 {
	return burtCoin.Balance(user)
}

func (bc *BurtCoin) saveWalletsToFile() {
	bc.lock.Lock()
	defer bc.lock.Unlock()
	json, err := json.Marshal(bc.Wallets)
	if err != nil {
		log.Println("Couldn't json")
		return
	}
	if err := os.WriteFile("./wallets.json", json, 0644); err != nil {
		log.Println(err.Error())
	}
}

func (bc *BurtCoin) Help() []string {
	return []string{
		"This is all very pointless...",
		"!burtcoin mine start|stop to start or stop a miner",
		"!burtcoin balance to check your current balance",
		"!burtcoin give [username] [amount] to give someone some useless burtcoin",
		"Waste everyones time.",
	}
}
