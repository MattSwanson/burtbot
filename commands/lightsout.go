package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gempir/go-twitch-irc/v2"
)

type LightsOut struct {
	CommChannel chan string
}

func (l *LightsOut) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if len(args) < 2 {
		return
	}
	if args[1] == "start" || args[1] == "stop" || args[1] == "reset" {
		l.CommChannel <- fmt.Sprintf("lo %s", args[1])
		return
	}
	if n, err := strconv.Atoi(args[1]); err == nil {
		l.CommChannel <- fmt.Sprintf("lo %d", n)
	}
}

func (l *LightsOut) Init() {

}

func (l *LightsOut) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {

}

func (l *LightsOut) Help() []string {
	return []string{
		"!lo start|stop to begin or end the game",
		"!lo [number] to press the light at the corresponding position of the board",
		"!lo reset will ... reset the game",
	}
}
