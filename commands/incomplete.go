package commands

import (
	"fmt"
	"strings"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type Incomplete struct {
	count int
}

var incomplete *Incomplete = &Incomplete{}

func init() {
	RegisterCommand("incomplete", incomplete)
}

func (i *Incomplete) Run(msg twitch.PrivateMessage) {
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if len(args) <= 1 {
		i.count++
		comm.ToChat(msg.Channel, "Oh, another uhh...")
		return
	}
	if args[1] == "count" {
		comm.ToChat(msg.Channel, fmt.Sprintf("Been lost in thought %d times today.", i.count))
	}
}

func (i *Incomplete) Init() {

}

func (i Incomplete) Help() []string {
	return []string{
		"!incomplete to mark another point in time where we lose all track of what I can't remember",
		"!incomplete count to see how many times that happened?",
	}
}
