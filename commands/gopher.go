package commands

import (
	"strings"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type Gopher struct {}

var gopher *Gopher = &Gopher{}

func init() {
	RegisterCommand("go", gopher)
}

func (g *Gopher) Init() {

}

func (g *Gopher) Run(msg twitch.PrivateMessage) {
	//!go spawn
	//!go show
	//!go hide
	args := strings.Fields(strings.ToLower(msg.Message))
	if len(args) < 2 {
		return
	}
	if args[1] == "spawn" {
		n := "1"
		if len(args) >= 3 {
			n = args[2]
		}
		comm.ToOverlay("spawngo " + n)
		return
	}
	if args[1] == "kill" {
		comm.ToOverlay("killgophs")
		return
	}

}

func (g *Gopher) Help() []string {
	return []string{
		"!go spawn [number] will spawn some [number] of gophers",
		"!go kill will move all the gophers to another plane of existence.",
	}
}
