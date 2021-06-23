package commands

import (
	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type Nonillion struct {
}

func (n Nonillion) Init() {

}

func (n Nonillion) Run(msg twitch.PrivateMessage) {
	comm.ToChat(msg.Channel, "The cosmic microtone background becomes transparent")
}
