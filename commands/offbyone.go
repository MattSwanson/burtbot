package commands

import (
	"fmt"
	"strings"

	"github.com/gempir/go-twitch-irc/v2"
)

type OffByOneCounter struct {
	counter int
}

func (o *OffByOneCounter) Init() {

}

func (o *OffByOneCounter) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if len(args) == 2 {
		if args[1] == "count" {
			client.Say(msg.Channel, fmt.Sprintf("Off by one %d times today.", o.counter-1))
			return
		}
	}
	if len(args) > 1 {
		return
	}
	o.counter++
	client.Say(msg.Channel, fmt.Sprintf("Off by one again... we've been off by one %d times today.", o.counter-1))
}

func (o *OffByOneCounter) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {

}

func (o *OffByOneCounter) Help() []string {
	return []string{
		"!offbyone when they're off by one again",
		"!offbyone to see how many times we have been off by one",
	}
}
