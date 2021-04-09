package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gempir/go-twitch-irc/v2"
)

type Tanks struct {
	TcpChannel     chan string
	TwitchClient   *TwitchAuthClient
	running        bool
	currentPlayers []twitch.User
}

func (t *Tanks) Init() {

}

func (t *Tanks) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	args := strings.Fields(strings.ToLower(msg.Message))
	if len(args) < 2 {
		return
	}
	if args[1] == "start" && !t.running {
		t.TcpChannel <- "tanks start"
		t.running = true
	} else if args[1] == "stop" && t.running {
		t.TcpChannel <- "tanks stop"
		t.running = false
	} else if args[1] == "join" && t.running {
		u := t.TwitchClient.GetUser(msg.User.Name)
		t.TcpChannel <- fmt.Sprintf("tanks join %s %s", msg.User.DisplayName, u.ProfileImgURL)
	} else if args[1] == "reset" && t.running && isMod(msg.User) {
		t.TcpChannel <- "tanks reset"
	} else if args[1] == "shoot" && t.running {
		// args[2] will be angle in degrees - int
		// args[3] will be velo - float
		if len(args) < 4 {
			return
		}
		angle, err := strconv.Atoi(args[2])
		if err != nil || angle < 0 || angle > 360 {
			client.Say(msg.Channel, "Invalid angle")
			return
		}

		v, err := strconv.ParseFloat(args[3], 64)
		if err != nil || v <= 0 {
			client.Say(msg.Channel, "Invalid velocity")
			return
		}

		t.TcpChannel <- fmt.Sprintf("tanks shoot %s %d %.4f", msg.User.DisplayName, angle, v)
	} else if args[1] == "begin" {
		t.TcpChannel <- "tanks begin"
	}
}

func (t *Tanks) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {

}

func (t *Tanks) Stop() {
	t.running = false
}
