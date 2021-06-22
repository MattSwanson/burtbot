package commands

import (
	"strings"
	"fmt"

	"github.com/gempir/go-twitch-irc/v2"
)

type Wod struct{
	current string
}

func (w *Wod) Run(client *twitch.Client, msg twitch.PrivateMessage) {

	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if len(args) == 1 {
		// show the wod
		if w.current == "" {
			client.Say(msg.Channel, "There is no word of the day... what a boring day.")
			return
		}
		client.Say(msg.Channel, fmt.Sprintf("The word of the day is: %s", w.current))
	}	
	if len(args) > 2 && args[1] == "set" && IsMod(msg.User) {
		// set the wod
		w.current = args[2]
		client.Say(msg.Channel, fmt.Sprintf("The word of the day is now: %s", w.current))
	}
}

func (w *Wod) Init() {

}

func (w *Wod) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {

}

func (w *Wod) Help() []string {
	return []string{
		"!wod to see the word of the day",
		"!wod set to set the word of the day",
	}
}
