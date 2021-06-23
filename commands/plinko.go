package commands

import (
	"fmt"
	"strings"
	"strconv"
	"encoding/json"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type Plinko struct {
	TokenMachine *TokenMachine
	running      bool
}

func (p *Plinko) Init() {
	comm.SubscribeToReply("plinko", p.HandleResponse)
	comm.SubscribeToReply("reset", p.Stop)
}

func (p *Plinko) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	if p.TokenMachine == nil {
		p.TokenMachine = getTokenMachine()
	}
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
		if args[2] == "all" && numTokens >= 5 {
			cost = 5
		}
		p.TokenMachine.setTokenCount(msg.User.Name, numTokens-cost)
		comm.ToOverlay(fmt.Sprintf("plinko drop %s %s %s", args[2], msg.User.DisplayName, msg.User.Color))
	}

}

func (p *Plinko) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {
	// lame
}

func (p *Plinko) HandleResponse(args []string) {
	if n, err := strconv.Atoi(args[3]); err == nil {
		p.TokenMachine.GrantToken(strings.ToLower(args[2]), n)

		s := ""
		if n > 0 {
			plural := ""
			if n > 1 {
				plural = "s"
			}
			s = fmt.Sprintf("@%s won %d token%s!", args[2], n, plural)
		} else {
			s = fmt.Sprintf("@%s, YOU GET NOTHING! GOOD DAY!", args[2])
		}
		//client.Say("burtstanton", s)
		mMsg := MarqueeMsg{
			RawMessage: s,
			Emotes:     "",
		}
		json, err := json.Marshal(mMsg)
		if err != nil {
			return
		}
		comm.ToOverlay("marquee once " + string(json))
	}
}

func (p *Plinko) Stop(args []string) {
	//comm.ToOverlay("plinko stop")
	p.running = false
}

func (p *Plinko) Help() []string {
	return []string{
		"!plinko drop [number] to drop a token at the specified drop point",
		"!plinko drop all will drop a token at each drop point",
		"Each token costs one token.",
	}
}
