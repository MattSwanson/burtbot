package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type fs struct{}

var dfs *fs = &fs{}

func init() {
	RegisterCommand("fs", dfs)
}

func (f *fs) PostInit() {}

func (f *fs) Run(msg twitch.PrivateMessage) {
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if len(args) < 2 {
		return
	}

	switch args[1] {
	case "data":
		comm.ToOverlay("fsToggle")
	case "cam":
		if len(args) < 3 {
			comm.ToChat(msg.Channel, "Need to provide a camera number")
			return
		}
		n, err := strconv.Atoi(args[2])
		if err != nil {
			comm.ToChat(msg.Channel, "Invalid camera number")
			return
		}
		comm.ToOverlay(fmt.Sprintf("fs camera %d", n))
	case "alt":
		if len(args) < 3 || !IsMod(msg.User) {
			return
		}
		n, err := strconv.Atoi(args[2])
		if err != nil || n < 0 {
			return
		}
		comm.ToOverlay(fmt.Sprintf("fs alt %d", n))
	case "navlights":
		comm.ToOverlay("fs navlights")
	case "e1f":
		if !IsMod(msg.User) {
			return
		}
		comm.ToOverlay("fs eng1f")
	case "autopilot":
		if len(args) < 3 || (args[2] != "off" && args[2] != "on") {
			comm.ToChat(msg.Channel, "Needs to be off or on")
			return
		}
		comm.ToOverlay(fmt.Sprintf("fs autopilot %s", args[2]))
	case "hdg":
		if len(args) < 3 {
			return
		}
		n, err := strconv.Atoi(args[2])
		if err != nil || n < 0 || n > 359 {
			return
		}
		comm.ToOverlay(fmt.Sprintf("fs hdg %d", n))
	case "togglelc":
		comm.ToOverlay("fs togglelc")
	case "lcset":
		if !IsMod(msg.User) || len(args) < 3 {
			return
		}
		n, err := strconv.Atoi(args[2])
		if err != nil || n < 0 {
			return
		}
		comm.ToOverlay(fmt.Sprintf("fs lc %d", n))
	}
}

func (f *fs) Help() []string {
	return []string{
		"!fs - Toggle the flight sim info on/off",
	}
}
