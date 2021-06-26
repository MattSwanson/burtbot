package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type Marquee struct {}

type MarqueeMsg struct {
	RawMessage string `json:"rawMessage"`
	Emotes     string `json:"emotes"`
}

var marquee *Marquee = &Marquee{}

func init() {
	RegisterCommand("marquee", marquee)
}

func (n *Marquee) PostInit() {

}

func (n *Marquee) Run(msg twitch.PrivateMessage) {
	// if !isMod(msg.User) {
	// 	return
	// }
	args := strings.Fields(msg.Message)
	if len(args) < 2 {
		return
	}
	if args[1] == "off" {
		comm.ToOverlay("marquee off")
	}
	var offset int
	if args[1] == "set" {
		offset = 13
	} else if args[1] == "once" {
		offset = 14
	} else {
		return
	}
	mMsg := MarqueeMsg{
		RawMessage: msg.Message[offset:],
		Emotes:     msg.Tags["emotes"],
	}
	j, err := json.Marshal(mMsg)
	if err != nil {
		log.Println(err.Error())
		return
	}
	comm.ToOverlay(fmt.Sprintf("marquee %s %s", args[1], string(j)))
}

func (m *Marquee) Help() []string {
	return []string{
		"!marquee set [text] will create a scrolling marquee with the given text",
		"!marquee once [text] will do the same but it will only go across once",
		"!marquee off to turn them all off and ruin everyone's fun.",
	}
}
