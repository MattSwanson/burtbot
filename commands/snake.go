package commands

import (
	"strings"

	"github.com/gempir/go-twitch-irc/v2"
)

type Snake struct {
	TcpChannel chan string
	isRunning  bool
}

func (s *Snake) Init() {

}

func (s *Snake) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	args := strings.Fields(strings.ToLower(msg.Message))
	if len(args) < 2 || args[1] == "start" {
		if !s.isRunning {
			s.TcpChannel <- "snake start"
			s.isRunning = true
		}
	} else if len(args) >= 2 && args[1] == "stop" {
		if s.isRunning {
			s.TcpChannel <- "snake stop"
			s.isRunning = false
		}
	}
}

func (s *Snake) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {

}
