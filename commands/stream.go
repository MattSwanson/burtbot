package commands

import (
	"strings"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type Stream struct{}

var s *Stream = &Stream{}

func init() {
	RegisterCommand("stream", s)
}

func (cs *Stream) Run(msg twitch.PrivateMessage) {
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if !IsMod(msg.User) || len(args) < 2 {
		return
	}
	if args[1] == "start" {
		comm.ToOverlay("stream start")
	}
	if args[1] == "stop" {
		comm.ToOverlay("stream stop")
	}
}

func (cs *Stream) PostInit() {

}

func (cs *Stream) Help() []string {
	return []string{
		"",
	}
}
