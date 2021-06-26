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
	running      bool
}

var plinko *Plinko = &Plinko{}

func init() {
	comm.SubscribeToReply("plinko", plinko.HandleResponse)
	comm.SubscribeToReply("reset", plinko.Stop)
	RegisterCommand("plinko", plinko)
}

func (p *Plinko) PostInit() {

}

func (p *Plinko) Run(msg twitch.PrivateMessage) {
	args := strings.Fields(strings.ToLower(msg.Message))
	if len(args) < 2 {
		return
	}

	if args[1] == "drop" && len(args) >= 3 {
		numTokens := GetTokenCount(msg.User)
		if numTokens <= 0 {
			comm.ToChat(msg.Channel, fmt.Sprintf("Sorry @%s, you have no tokens. Plinko costs 1 token per drop.", msg.User.DisplayName))
			return
		}
		cost := 1
		if args[2] == "all" && numTokens >= 5 {
			cost = 5
		}
		DeductTokens(msg.User.Name, cost)
		comm.ToOverlay(fmt.Sprintf("plinko drop %s %s %s", args[2], msg.User.DisplayName, msg.User.Color))
	}

}

func (p *Plinko) HandleResponse(args []string) {
	if n, err := strconv.Atoi(args[3]); err == nil {
		GrantToken(strings.ToLower(args[2]), n)

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
		//comm.ToChat("burtstanton", s)
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
