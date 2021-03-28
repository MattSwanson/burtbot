package commands

import (
	"fmt"
	"strings"

	"github.com/gempir/go-twitch-irc/v2"
)

type Plinko struct {
	TcpChannel   chan string
	TokenMachine *TokenMachine
	running      bool
}

func (p *Plinko) Init() {

}

func (p *Plinko) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	args := strings.Fields(strings.ToLower(msg.Message))
	if len(args) < 2 {
		return
	}

	if args[1] == "start" && !p.running {
		p.running = true
		p.TcpChannel <- "plinko start"
		return
	}

	if args[1] == "stop" && p.running {
		if isMod(msg.User) {
			p.TcpChannel <- "plinko stop"
			p.running = false
			return
		}
	}

	// if p.currentPlayer == nil || p.currentPlayer.DisplayName != msg.User.DisplayName {
	// 	return
	// }
	// !plinko drop n username - username supplier by message not command, so len(args) = 3
	if args[1] == "drop" && len(args) >= 3 {
		numTokens := p.TokenMachine.getTokenCount(msg.User)
		if numTokens <= 0 {
			client.Say(msg.Channel, fmt.Sprintf("Sorry @%s, you have no tokens. Plinko costs 1 token per drop.", msg.User.DisplayName))
			return
		}
		cost := 1
		if args[2] == "all" && numTokens >= 5 {
			cost = 5
		}
		p.TokenMachine.setTokenCount(msg.User.Name, numTokens-cost)
		p.TcpChannel <- fmt.Sprintf("plinko drop %s %s", args[2], msg.User.DisplayName)
	}

	// switch args[1] {
	// // case "left":
	// // 	p.TcpChannel <- "plinko left"
	// // case "right":
	// // 	p.TcpChannel <- "plinko right"
	// case "drop":
	// 	p.TcpChannel <- "plinko drop"
	// }
}

func (p *Plinko) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {
	// lame
}

func (p *Plinko) Stop() {
	p.TcpChannel <- "plinko stop"
	p.running = false
}
