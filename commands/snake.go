package commands

import (
	"fmt"
	"strconv"
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
	} else if len(args) >= 2 && args[1] == "speed" {
		if s.isRunning && IsMod(msg.User) {
			n, err := strconv.Atoi(args[2])
			if err != nil {
				return
			}
			s.TcpChannel <- fmt.Sprintf("snake speed %d", n)
		}
	}
}

func (s *Snake) SetRunning(b bool) {
	s.isRunning = b
}

func (s *Snake) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {

}

func (s *Snake) Help() []string {
	return []string{
		"!snake start|stop starts or stops snake...",
		"Type w, a, s, d to move plinko around the screen",
		"Eat square apples and squeek",
	}
}
