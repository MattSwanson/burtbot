package commands

import (
	"strings"

	"github.com/gempir/go-twitch-irc/v2"
)

type BigMouse struct {
	TcpChannel chan string
}

func (m *BigMouse) Init() {

}

func (m *BigMouse) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	args := strings.Fields(strings.ToLower(msg.Message))
	if len(args) < 2 {
		return
	}
	if args[1] == "on" {
		m.TcpChannel <- "bigmouse true"
		return
	}
	if args[1] == "off" {
		m.TcpChannel <- "bigmouse false"
		return
	}
}

func (m *BigMouse) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {

}

func (m *BigMouse) Help() []string {
	return []string{
		"!bigmouse on|off to enable/disable big mouse mode [NOT WORKING RIGHT NOW... lazy]",
	}
}
