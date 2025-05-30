package commands

import (
	"fmt"
	"time"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type Tim struct{}

var tim *Tim = &Tim{}

func init() {
	RegisterCommand("time", tim)
}

func (t *Tim) PostInit() {

}

func (t Tim) Run(msg twitch.PrivateMessage) {
	ctime := time.Now().Unix()
	comm.ToChat(msg.Channel, fmt.Sprintf("The time is now %d", ctime))
}

func (t *Tim) Help() []string {
	return []string{
		"What time is it Mr. Clock?",
	}
}
