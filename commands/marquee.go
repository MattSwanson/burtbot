package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/gempir/go-twitch-irc/v2"
)

type Marquee struct {
	TcpChannel chan string
}

type MarqueeMsg struct {
	RawMessage string `json:"rawMessage"`
	Emotes     string `json:"emotes"`
}

func (n *Marquee) Init() {

}

func (n *Marquee) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	// if !isMod(msg.User) {
	// 	return
	// }
	args := strings.Fields(msg.Message)
	if len(args) < 2 {
		return
	}
	// msg.Tags["emotes"]
	fmt.Println(msg.Emotes)
	fmt.Println(msg.Tags["emotes"])
	mMsg := MarqueeMsg{
		RawMessage: strings.Join(args[2:], " "),
		Emotes:     msg.Tags["emotes"],
	}
	j, err := json.Marshal(mMsg)
	if err != nil {
		log.Println(err.Error())
		return
	}
	if args[1] == "off" {
		n.TcpChannel <- "marquee off"
	} else if args[1] == "set" {
		n.TcpChannel <- "marquee set " + string(j)
	} else if args[1] == "once" {
		n.TcpChannel <- "marquee once " + string(j)
	}

	/*else if args[1] == "embiggen" {
		n.TcpChannel <- "setmarquee " + "embiggen"
	} else if args[1] == "smol" {
		n.TcpChannel <- "setmarquee " + "smol"
	}*/
}

func (n *Marquee) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {

}

func (m *Marquee) Help() []string {
	return []string{
		"!marquee set [text] will create a scrolling marquee with the given text",
		"!marquee once [text] will do the same but it will only go across once",
		"!marquee off to turn them all off and ruin everyone's fun.",
	}
}
