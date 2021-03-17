package commands

import (
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gempir/go-twitch-irc/v2"
)

type Msg struct{}

func (m *Msg) Init() {

}

func (m *Msg) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	if !isMod(msg.User) {
		return
	}
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if len(args) < 2 {
		client.Say(msg.Channel, "Not enough stuff for stuff")
		return
	}
	newMsg := strings.Join(args[1:], " ")
	_, err := http.PostForm("http://localhost:8080/sendMessage", url.Values{"message": {newMsg}})
	if err != nil {
		client.Say(msg.Channel, "cud not message")
		log.Println(err.Error())
		return
	}
}

func (m *Msg) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {
	return
}
