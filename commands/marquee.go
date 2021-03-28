package commands

import (
	"strings"

	"github.com/gempir/go-twitch-irc/v2"
)

type Marquee struct {
	TcpChannel chan string
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
	if args[1] == "set" {
		n.TcpChannel <- "marquee set " + strings.Join(args[2:], " ")
	} else if args[1] == "once" {
		n.TcpChannel <- "marquee once " + strings.Join(args[2:], " ")
	}

	/*else if args[1] == "embiggen" {
		n.TcpChannel <- "setmarquee " + "embiggen"
	} else if args[1] == "smol" {
		n.TcpChannel <- "setmarquee " + "smol"
	}*/
}

func (n *Marquee) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {

}
