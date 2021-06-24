package commands

import (
	"fmt"
	"strings"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type OffByOneCounter struct {
	counter int
}

var obo *OffByOneCounter = &OffByOneCounter{}

func init() {
	RegisterCommand("offbyone", obo)
}

func (o *OffByOneCounter) Init() {

}

func (o *OffByOneCounter) Run(msg twitch.PrivateMessage) {
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if len(args) == 2 {
		if args[1] == "count" {
			comm.ToChat(msg.Channel, fmt.Sprintf("Off by one %d times today.", o.counter-1))
			return
		}
	}
	if len(args) > 1 {
		return
	}
	o.counter++
	comm.ToChat(msg.Channel, fmt.Sprintf("Off by one again... we've been off by one %d times today.", o.counter-1))
}

func (o *OffByOneCounter) Help() []string {
	return []string{
		"!offbyone when they're off by one again",
		"!offbyone to see how many times we have been off by one",
	}
}
