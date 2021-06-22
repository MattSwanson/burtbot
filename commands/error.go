package commands

import (
	"github.com/gempir/go-twitch-irc/v2"
)

type ErrorBox struct {
	TcpChannel chan string
}

func (e ErrorBox) Init() {

}

func (e *ErrorBox) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	e.TcpChannel <- "error"
}

func (e ErrorBox) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {

}

func (e ErrorBox) Help() []string {
	return []string{}
}
