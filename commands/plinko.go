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

	if args[1] == "drop" && len(args) >= 3 {
		numTokens := p.TokenMachine.getTokenCount(msg.User)
		if numTokens <= 0 {
			client.Say(msg.Channel, fmt.Sprintf("Sorry @%s, you have no tokens. Plinko costs 1 token per drop.", msg.User.DisplayName))
			return
		}
		cost := 1
		if args[2] == "all" && numTokens >= 9 {
			cost = 9
		}
		p.TokenMachine.setTokenCount(msg.User.Name, numTokens-cost)
		p.TcpChannel <- fmt.Sprintf("plinko drop %s %s %s", args[2], msg.User.DisplayName, msg.User.Color)
	}

}

func (p *Plinko) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {
	// lame
}

func (p *Plinko) Stop() {
	p.TcpChannel <- "plinko stop"
	p.running = false
}

func (p *Plinko) Help() []string {
	return []string{
		"!plinko drop [number] to drop a token at the specified drop point",
		"!plinko drop all will drop a token at each drop point",
		"Each token costs one token.",
	}
}
