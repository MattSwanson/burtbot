package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type Plinko struct {
	running bool
}

const (
	superPlinkoChance     = 100
	superPlinkoMultiplier = 5
)

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

	userColor := msg.User.Color
	if userColor == "" {
		userColor = "#0000FF"
	}

	if args[1] == "auto" && IsMod(msg.User) {
		if autoCancel != nil {
			autoCancel()
			autoCancel = nil
			return
		}
		var ctx context.Context
		ctx, autoCancel = context.WithCancel(context.Background())
		go func(ctx context.Context, user twitch.User, channel string) {
			for {
				select {
				case <-ctx.Done():
					comm.ToChat(channel, "Stopping auto plinko")
					return
				default:
					r := rand.Intn(5)
					tokenCount := GetTokenCount(user)
					if tokenCount.Cmp(big.NewInt(0)) == -1 || tokenCount.Cmp(big.NewInt(0)) == 0 {
						comm.ToChat(channel, "You're out of tokens, stopping auto plinko")
						return
					}
					comm.ToOverlay(fmt.Sprintf("plinko drop %d %s %s", r, user.DisplayName, userColor))
				}
				time.Sleep(time.Second * 5)
			}
		}(ctx, msg.User, msg.Channel)
	}

	if args[1] == "drop" && len(args) >= 3 {
		numTokens := GetTokenCount(msg.User)
		if numTokens.Cmp(big.NewInt(0)) == -1 || numTokens.Cmp(big.NewInt(0)) == 0 {
			comm.ToChat(msg.Channel, fmt.Sprintf("Sorry @%s, you have no tokens. Plinko costs 1 token per drop.", msg.User.DisplayName))
			return
		}
		cost := 1
		drop, err := strconv.Atoi(args[2])
		if err != nil || drop < 0 || drop > 4 {
			comm.ToChat(msg.Channel, fmt.Sprintf("Invalid drop zone specified for Plinko. Valid drop zones are 0, 1, 2, 3, or 4"))
			return
		}
		DeductTokens(msg.User.Name, big.NewInt(int64(cost)))
		if rand.Intn(superPlinkoChance) == 0 {
			cost *= superPlinkoMultiplier
			comm.ToChat(msg.Channel, fmt.Sprintf("WOW, %s got a Super Plinko token worth 5x! Good luck!", msg.User.DisplayName))
		}
		comm.ToOverlay(fmt.Sprintf("plinko drop %s %s %s %d", args[2], msg.User.DisplayName, userColor, cost))
		return
	}

	if args[1] == "super" {
		comm.ToChat(msg.Channel, fmt.Sprintf("@%s: It is with much regret and sadness that I must inform you that Super Plinko has been disabled. It does live on in other ways though.", msg.User.DisplayName))
	}
}

func (p *Plinko) HandleResponse(args []string) {
	n := big.NewInt(0)
	_, err := fmt.Sscan(args[3], n)
	if err != nil {
		log.Println("Couldn't parse number from overlay response", err)
		return
	}
	GrantToken(strings.ToLower(args[2]), n)
	s := ""
	if n.Cmp(big.NewInt(0)) == 1 {
		plural := ""
		if n.Cmp(big.NewInt(1)) == 1 {
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

func (p *Plinko) Stop(args []string) {
	//comm.ToOverlay("plinko stop")
	p.running = false
}

func (p *Plinko) Help() []string {
	return []string{
		"!plinko drop [number] to drop a token at the specified drop point",
		"!plinko drop all will drop a token at each drop point",
		"Each token costs one token.",
		"1 in 100 chance to get a Super Plinko token worth 5x!",
		//	"!plinko super [number] [wager] will drop a token for the wagered amount",
	}
}
