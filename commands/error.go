package commands

import (
	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type ErrorBox struct {}

func (e ErrorBox) Init() {

}

func (e *ErrorBox) Run(msg twitch.PrivateMessage) {
	comm.ToOverlay("error")
}

func (e ErrorBox) Help() []string {
	return []string{}
}
