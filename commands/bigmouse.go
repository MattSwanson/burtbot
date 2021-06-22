package commands

import (
	"github.com/gempir/go-twitch-irc/v2"
)

type BigMouse struct {
	TcpChannel chan string
}

func (m *BigMouse) Init() {

}

func (m *BigMouse) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	m.TcpChannel <- "bigmouse"
}

func (m *BigMouse) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {

}

func (m *BigMouse) Help() []string {
	return []string{
		"!bigmouse to toggle big mouse mode [NOT WORKING RIGHT NOW... lazy]",
	}
}
