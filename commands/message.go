package commands

import (
	"fmt"
	"strings"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type Msg struct{}

var msg *Msg = &Msg{}

func init() {
	RegisterCommand("bbmsg", msg)
}

func (m *Msg) PostInit() {

}

func (m *Msg) Run(msg twitch.PrivateMessage) {
	if !IsMod(msg.User) {
		return
	}
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if len(args) < 2 {
		comm.ToChat(msg.Channel, "Not enough stuff for stuff")
		return
	}
	newMsg := strings.Join(args[1:], " ")
	comm.ToOverlay(fmt.Sprintf("tts false false %s", newMsg))
}

func (m *Msg) Help() []string {
	return []string{
		"!bbmsg [text] to make me say the thing",
	}
}
