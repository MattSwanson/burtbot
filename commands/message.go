package commands

import (
	"fmt"
	"strings"

	"github.com/gempir/go-twitch-irc/v2"
)

type Msg struct {
	TcpChannel chan string
}

func (m *Msg) Init() {

}

func (m *Msg) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	// if !isMod(msg.User) {
	// 	return
	// }
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if len(args) < 2 {
		client.Say(msg.Channel, "Not enough stuff for stuff")
		return
	}
	newMsg := strings.Join(args[1:], " ")
	m.TcpChannel <- fmt.Sprintf("tts false %s", newMsg)
}

func (m *Msg) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {

}

func (m *Msg) Help() []string {
	return []string{
		"!bbmsg [text] to make me say the thing",
	}
}
