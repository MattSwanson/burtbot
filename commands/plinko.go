package commands

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"strconv"
	"time"
	"encoding/json"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type Plinko struct {
	running      bool
}

var plinko *Plinko = &Plinko{}
var autoCancel context.CancelFunc
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

	if args[1] == "auto" && IsMod(msg.User) {
		if autoCancel != nil {
			autoCancel()
			autoCancel = nil
			return
		}
		var ctx context.Context
		ctx, autoCancel = context.WithCancel(context.Background())
		go func(ctx context.Context, user twitch.User, channel string){
			for {
				select {
					case <-ctx.Done():
						comm.ToChat(channel, "Stopping auto plinko")
						return
					default:
						r := rand.Intn(5)
						if GetTokenCount(user) <= 0 {
							comm.ToChat(channel, "You're out of tokens, stopping auto plinko")
							return
						}
						comm.ToOverlay(fmt.Sprintf("plinko drop %d %s %s", r, user.DisplayName, user.Color))
				}
				time.Sleep(time.Second * 5)
			}
		}(ctx, msg.User, msg.Channel)
	}

	if args[1] == "drop" && len(args) >= 3 {
		numTokens := GetTokenCount(msg.User)
		if numTokens <= 0 {
			comm.ToChat(msg.Channel, fmt.Sprintf("Sorry @%s, you have no tokens. Plinko costs 1 token per drop.", msg.User.DisplayName))
			return
		}
		cost := 1
		drop, err := strconv.Atoi(args[2])
		if err != nil {
			if args[2] == "all" && numTokens >= 5 {
				cost = 5
			} else {
				return
			}
		}
		if drop < 0 || drop > 4 {
			return
		}
		DeductTokens(msg.User.Name, uint64(cost))
		comm.ToOverlay(fmt.Sprintf("plinko drop %s %s %s", args[2], msg.User.DisplayName, msg.User.Color))
		return
	}

	if args[1] == "super" && len(args) >= 4 {
		n, err := strconv.ParseUint(args[3], 10, 64)
		if err != nil || n < 0 {
			return
		}
		drop, err := strconv.Atoi(args[2])
		if err != nil || drop < 0 || drop > 4 {
			return
		}
		if count := GetTokenCount(msg.User); count < n {
			comm.ToChat(msg.Channel, fmt.Sprintf("@%s, you only have %d tokens. Can't wager %d.", msg.User.DisplayName, count, n))
			return
		}
		DeductTokens(msg.User.Name, n)
		comm.ToOverlay(fmt.Sprintf("plinko drop %s %s %s %d", args[2], msg.User.DisplayName, msg.User.Color, n))
	}
}

func (p *Plinko) HandleResponse(args []string) {
	if n, err := strconv.ParseUint(args[3], 10, 64); err == nil {
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
		"!plinko super [number] [wager] will drop a token for the wagered amount",
	}
}
