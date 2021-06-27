package commands

import (
	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type ErrorBox struct {}
var errorBox *ErrorBox = &ErrorBox{}

func init() {
	RegisterCommand("error", errorBox)
}

func (e ErrorBox) PostInit() {

}

func (e *ErrorBox) Run(msg twitch.PrivateMessage) {
	comm.ToOverlay("error")
}

func (e ErrorBox) Help() []string {
	return []string{"Sorry for the inconvenience."}
}
