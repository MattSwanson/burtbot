package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type Snake struct {
	isRunning  bool
}

func (s *Snake) Init() {
	comm.SubscribeToReply("reset", s.Stop)
}

func (s *Snake) Run(msg twitch.PrivateMessage) {
	args := strings.Fields(strings.ToLower(msg.Message))
	if len(args) < 2 || args[1] == "start" {
		if !s.isRunning {
			comm.ToOverlay("snake start")
			s.isRunning = true
		}
	} else if len(args) >= 2 && args[1] == "stop" {
		if s.isRunning {
			comm.ToOverlay("snake stop")
			s.isRunning = false
		}
	} else if len(args) >= 2 && args[1] == "speed" {
		if s.isRunning && IsMod(msg.User) {
			n, err := strconv.Atoi(args[2])
			if err != nil {
				return
			}
			comm.ToOverlay(fmt.Sprintf("snake speed %d", n))
		}
	}
}

func (s *Snake) SetRunning(b bool) {
	s.isRunning = b
}

func (s *Snake) Stop(args []string) {
	s.isRunning = false
}

func (s *Snake) Help() []string {
	return []string{
		"!snake start|stop starts or stops snake...",
		"Type w, a, s, d to move plinko around the screen",
		"Eat square apples and squeek",
	}
}
