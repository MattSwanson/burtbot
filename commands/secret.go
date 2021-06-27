package commands

import (
	"fmt"
	"time"
	"strings"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

var lastQuacksplosion time.Time
var lastMessage string
var lastMsg twitch.PrivateMessage 
var schlorpLock = false
var schlorpCD = 10

func init() {
	SubscribeToRawMsg(secretCommands)
}
// Parse raw message here for secret commands
// Don't edit these on stream...
func secretCommands(msg twitch.PrivateMessage) {
	
	if msg.User.DisplayName == "tundragaminglive" {
		comm.ToOverlay("miracle")
	}

	lower := strings.ToLower(msg.Message)
	if strings.Contains(lower, "one time") {
		comm.ToChat(msg.Channel, "ONE TIME!")
	}
	if count := strings.Count(lower, "quack"); count > 0 {
		comm.ToOverlay(fmt.Sprintf("quack %d", count))
		if msg.User.DisplayName == "0xffffffff810000000" {
			if time.Since(lastQuacksplosion).Seconds() > 21600 {
				comm.ToOverlay("quacksplosion")
				lastQuacksplosion = time.Now()
			}
		}
	}
	
	if strings.Compare(msg.User.Name, lastMsg.User.Name) == 0 && strings.Compare(msg.Message, lastMessage+" "+lastMessage) == 0 {
		// break the pyramid with a schlorp
		comm.ToChat(msg.Channel, "tjportSchlorp1 tjportSchlorp2 tjportSchlorp3")
	}
	lower = strings.ToLower(msg.Message)
	if strings.Contains(lower, "schlorp") {
		if !schlorpLock {
			schlorpLock = true
			go unlockSchlorp()
			comm.ToChat(msg.Channel, "tjportSchlorp1 tjportSchlorp2 tjportSchlorp3")
		}
	}

	lastMessage = msg.Message
	lastMsg = msg
}

func unlockSchlorp() {
	time.Sleep(time.Second * time.Duration(schlorpCD))
	schlorpLock = false
}
