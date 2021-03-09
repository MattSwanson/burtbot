package commands

import (
	"github.com/gempir/go-twitch-irc/v2"
)

type Nonillion struct {
}

func (n Nonillion) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	client.Say(msg.Channel, "The cosmic microtone background becomes transparent")
}
