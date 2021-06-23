package commands

import (
	"fmt"
	"time"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type Tim struct{}

func (t *Tim) Init() {

}

func (t Tim) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	ctime := time.Now().Unix()
	comm.ToChat(msg.Channel, fmt.Sprintf("The time is now %d", ctime))
}

func (t *Tim) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {

}

func (t *Tim) Help() []string {
	return []string{
		"What time is it Mr. Clock?",
	}
}
