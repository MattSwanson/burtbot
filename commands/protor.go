package commands

import "github.com/gempir/go-twitch-irc/v2"

type ProtoR struct{}

func (p *ProtoR) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	client.Say(msg.Channel, "Check out my game on Youtube! https://youtu.be/dQw4w9WgXcQ")
}

func (p *ProtoR) Init() {

}

func (p *ProtoR) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {

}

func (p *ProtoR) Help() []string {
	return []string{
		"You're on your own here...",
	}
}
