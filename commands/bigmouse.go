package commands

import (
	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type BigMouse struct {}

var bigMouse *BigMouse = &BigMouse{}

func init() {
	RegisterCommand("bigmouse", bigMouse)
}

func (m *BigMouse) Init() {

}

func (m *BigMouse) Run(msg twitch.PrivateMessage) {
	comm.ToOverlay("bigmouse")
}

func (m *BigMouse) Help() []string {
	return []string{
		"!bigmouse to toggle big mouse mode [NOT WORKING RIGHT NOW... lazy]",
	}
}
