package commands

import (
	"fmt"
	"strings"

	"github.com/gempir/go-twitch-irc/v2"
)

type Gopher struct {
	TcpChannel chan string
}

func (g *Gopher) Init() {

}

func (g *Gopher) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	//!go spawn
	//!go show
	//!go hide
	args := strings.Fields(strings.ToLower(msg.Message))
	if len(args) < 2 {
		return
	}
	if args[1] == "spawn" {
		n := "1"
		if len(args) == 3 {
			n = args[2]
		}
		fmt.Println("spawn a goph")
		g.TcpChannel <- "spawngo " + n
		return
	}
	if args[1] == "hide" {
		fmt.Println("hide goph")
		g.TcpChannel <- "hidego"
		return
	}
	if args[1] == "show" {
		fmt.Println("show goph")
		g.TcpChannel <- "showgo"
		return
	}
	if args[1] == "size" {
		if len(args) < 3 {
			return
		}
		g.TcpChannel <- "sizego " + args[2]
		return
	}
	if args[1] == "kill" {
		g.TcpChannel <- "killgophs"
		return
	}

}

func (g *Gopher) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {

}
