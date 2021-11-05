package commands

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

var lastQuacksplosion time.Time
var lastMK time.Time
var lastArrowMsg time.Time
var lastMiracle time.Time
var lastTux time.Time
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

	if msg.User.DisplayName == "tundragaminglive" &&
		time.Since(lastMiracle).Seconds() > 21600 {
		comm.ToOverlay("miracle")
		lastMiracle = time.Now()
	}

	name := strings.ToLower(msg.User.DisplayName)
	if name == "somecodingguy" &&
		time.Since(lastMK).Seconds() > 21600 {
		if rand.Intn(1000) >= 900 {
			comm.ToOverlay("mk")
			lastMK = time.Now()
		}
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

	if strings.Contains(lower, "tux") {
		if time.Since(lastTux).Seconds() > 300 {
			comm.ToOverlay("tux")
			lastTux = time.Now()
		}
	}

	if strings.HasPrefix(lower, "!real") {
		args := strings.Fields(msg.Message)
		if len(args) != 2 {
			return
		}
		comm.ToChat(msg.Channel, fmt.Sprintf("%s is the only real language", args[1]))
	}

	if strings.ToLower(msg.User.DisplayName) == "velusip" {
		/*	if time.Since(lastArrowMsg).Seconds() > 21600 {
			comm.ToChat(msg.Channel, " Arrow keys all day -> -> -> -> -> -> -> -> -> -> -> ")
			comm.ToOverlay("tts true For absolutely no reason, everyone hold down your right arrow key for a very long time")
			m := MarqueeMsg{
				RawMessage: " -> -> -> -> -> -> -> -> -> -> -> -> -> -> -> -> -> -> -> -> -> -> -> -> -> -> -> -> -> ",
			}
			j, err := json.Marshal(m)
			if err != nil {
				return
			}
			comm.ToOverlay(fmt.Sprintf("marquee once %s", string(j)))
			lastArrowMsg = time.Now()
		}*/
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
