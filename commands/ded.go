package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type Ded struct{}
type counter struct {
	Count int
}

var cooldown int = 15
var locked = false
var count int
var ded *Ded = &Ded{}

func init() {
	RegisterCommand("ded", ded)
}

func (d *Ded) PostInit() {

}

func (d *Ded) Run(msg twitch.PrivateMessage) {
	if locked && !IsMod(msg.User) {
		return
	}
	// Mods can run this during cooldown - but don't elongate the cooldown
	if !locked {
		locked = true
		go unlock()
	}
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if len(args) > 2 {
		comm.ToChat(msg.Channel, "Too many arguments to ded. Why you do dis?")
		return
	}
	if len(args) == 2 {
		if !IsMod(msg.User) {
			comm.ToChat(msg.Channel, "Only mods can set the counter directly.")
			return
		}
		newCount, err := strconv.Atoi(args[1])
		if err != nil {
			comm.ToChat(msg.Channel, "ded requires a number not a thing else")
			return
		}
		count = newCount
		comm.ToChat(msg.Channel, fmt.Sprintf("ded counter set to %d", count))
		comm.ToOverlay(fmt.Sprintf("ded %d", count))
		return
	}

	count++
	var plural string
	if count > 1 {
		plural = "s"
	}
	comm.ToChat(msg.Channel, fmt.Sprintf("Has ded %d time%s.", count, plural))
	if count == 1 {
		time.Sleep(time.Millisecond * time.Duration(1000))
		comm.ToChat(msg.Channel, "ONE TIME!")
	}
	comm.ToOverlay(fmt.Sprintf("ded %d", count))
}

func unlock() {
	time.Sleep(time.Second * time.Duration(cooldown))
	locked = false
}

func (d *Ded) Help() []string {
	return []string{
		"!ded to increment the ded counter because streamer is bad at game",
	}
}
