package commands

import (
	"fmt"
	"time"

	"github.com/gempir/go-twitch-irc/v2"
)

type Tim struct{}

func (t *Tim) Init() {

}

func (t Tim) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	ctime := time.Now().Unix()
	client.Say(msg.Channel, fmt.Sprintf("The time is now %d", ctime))
}
