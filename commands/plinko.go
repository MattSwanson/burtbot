package commands

import (
	"context"
	"fmt"
	"log"
	"math/big"
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
						tokenCount := GetTokenCount(user)
						if tokenCount.Cmp(big.NewInt(0)) == -1 || tokenCount.Cmp(big.NewInt(0)) == 0 {
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
		if numTokens.Cmp(big.NewInt(0)) == -1 || numTokens.Cmp(big.NewInt(0)) == 0 {
			comm.ToChat(msg.Channel, fmt.Sprintf("Sorry @%s, you have no tokens. Plinko costs 1 token per drop.", msg.User.DisplayName))
			return
		}
		cost := 1
		drop, err := strconv.Atoi(args[2])
		if err != nil {
			if args[2] == "all" && numTokens.Cmp(big.NewInt(5)) == 1 || numTokens.Cmp(big.NewInt(5)) == 0 {
				cost = 5
			} else {
				return
			}
		}
		if drop < 0 || drop > 4 {
			return
		}
		DeductTokens(msg.User.Name, big.NewInt(int64(cost)))
		comm.ToOverlay(fmt.Sprintf("plinko drop %s %s %s", args[2], msg.User.DisplayName, msg.User.Color))
		return
	}

	if args[1] == "super" && len(args) >= 4 {
		n := big.NewInt(0)
		_, err := fmt.Sscan(args[3], n)
		if err != nil {
			comm.ToChat(msg.Channel, fmt.Sprintf("@%s, invalid token amount. Please try again", msg.User.DisplayName))
			return
		}
		drop, err := strconv.Atoi(args[2])
		if err != nil || drop < 0 || drop > 4 {
			return
		}
		if count := GetTokenCount(msg.User); count.Cmp(n) == -1 {
			comm.ToChat(msg.Channel, fmt.Sprintf("@%s, you only have %d tokens. Can't wager %d.", msg.User.DisplayName, count, n))
			return
		}
		DeductTokens(msg.User.Name, n)
		comm.ToOverlay(fmt.Sprintf("plinko drop %s %s %s %d", args[2], msg.User.DisplayName, msg.User.Color, n))
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
		"!plinko super [number] [wager] will drop a token for the wagered amount",
	}
}
