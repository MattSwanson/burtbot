package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/MattSwanson/burtbot/helix"
	"github.com/gempir/go-twitch-irc/v2"
)

type Tanks struct {
	running      bool
	//currentPlayers []twitch.User
}

var tanks *Tanks = &Tanks{}

func init() {
	RegisterCommand("tanks", tanks)
	comm.SubscribeToReply("reset", tanks.Stop)
}

func (t *Tanks) PostInit() {

}

func (t *Tanks) Run(msg twitch.PrivateMessage) {
	args := strings.Fields(strings.ToLower(msg.Message))
	if len(args) < 2 {
		return
	}
	if args[1] == "start" && !t.running {
		comm.ToOverlay("tanks start")
		t.running = true
	} else if args[1] == "stop" && t.running {
		comm.ToOverlay("tanks stop")
		t.running = false
	} else if args[1] == "join" && t.running {
		u := helix.GetUser(msg.User.Name)
		comm.ToOverlay(fmt.Sprintf("tanks join %s %s", msg.User.DisplayName, u.ProfileImgURL))
	} else if args[1] == "reset" && t.running && IsMod(msg.User) {
		comm.ToOverlay("tanks reset")
	} else if args[1] == "shoot" && t.running {
		// args[2] will be angle in degrees - int
		// args[3] will be velo - float
		if len(args) < 4 {
			return
		}
		angle, err := strconv.Atoi(args[2])
		if err != nil || angle < 0 || angle > 360 {
			comm.ToChat(msg.Channel, "Invalid angle")
			return
		}

		v, err := strconv.ParseFloat(args[3], 64)
		if err != nil || v <= 0 {
			comm.ToChat(msg.Channel, "Invalid velocity")
			return
		}

		comm.ToOverlay(fmt.Sprintf("tanks shoot %s %d %.4f", msg.User.DisplayName, angle, v))
	} else if args[1] == "begin" {
		comm.ToOverlay("tanks begin")
	}
}

func (t *Tanks) Stop(args []string) {
	t.running = false
}

func (t *Tanks) Help() []string {
	return []string{
		"!tanks start to load tanks",
		"!tanks join to join the game",
		"!tanks shoot [angle] [velocity] to shoot when it's your turn",
		"The angle is in degrees 0-360",
		"Velocity is a percentage 1 to 100",
	}
}
