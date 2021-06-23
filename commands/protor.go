package commands

import (
	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type ProtoR struct{}

func (p *ProtoR) Run(msg twitch.PrivateMessage) {
	comm.ToChat(msg.Channel, "Check out my game on Youtube! https://youtu.be/dQw4w9WgXcQ")
}

func (p *ProtoR) Init() {

}

func (p *ProtoR) Help() []string {
	return []string{
		"You're on your own here...",
	}
}
