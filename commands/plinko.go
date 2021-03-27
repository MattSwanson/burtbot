package commands

import (
	"strings"

	"github.com/gempir/go-twitch-irc/v2"
)

type Plinko struct {
	TcpChannel    chan string
	TokenMachine  *TokenMachine
	currentPlayer *twitch.User
}

func (p *Plinko) Init() {

}

func (p *Plinko) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	args := strings.Fields(strings.ToLower(msg.Message))
	if len(args) < 2 {
		return
	}

	if args[1] == "start" && p.currentPlayer == nil {
		numTokens := p.TokenMachine.getTokenCount(msg.User)
		if numTokens >= 1 {
			p.TokenMachine.setTokenCount(msg.User.Name, numTokens-1)
			p.currentPlayer = &msg.User
			p.TcpChannel <- "plinko start " + msg.User.DisplayName
			return
		}
	}

	if args[1] == "stop" && p.currentPlayer != nil {
		if isMod(msg.User) || p.currentPlayer.DisplayName == msg.User.DisplayName {
			p.TcpChannel <- "plinko stop"
			p.currentPlayer = nil
			return
		}
	}

	if p.currentPlayer == nil || p.currentPlayer.DisplayName != msg.User.DisplayName {
		return
	}
	switch args[1] {
	case "left":
		p.TcpChannel <- "plinko left"
	case "right":
		p.TcpChannel <- "plinko right"
	case "drop":
		p.TcpChannel <- "plinko drop"
	}
}

func (p *Plinko) ClearPlayer() {
	p.currentPlayer = nil
}

func (p *Plinko) GetPlayer() *twitch.User {
	return p.currentPlayer
}

func (p *Plinko) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {
	// lame
}

func (p *Plinko) Stop() {
	p.TcpChannel <- "plinko stop"
	p.currentPlayer = nil
}
