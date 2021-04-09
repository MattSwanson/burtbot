package commands

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/gempir/go-twitch-irc/v2"
)

type Shoutout struct {
	TcpChannel   chan string
	TwitchClient *TwitchAuthClient
}

func (s *Shoutout) Init() {
	rand.Seed(time.Now().UnixNano())
}

func (s *Shoutout) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	if !isMod(msg.User) {
		return
	}
	args := strings.Fields(strings.ToLower(msg.Message))
	if len(args) < 2 {
		return
	}
	u := s.TwitchClient.GetUser(args[1])
	fmt.Println(u)
	r := rand.Intn(100)
	if r < 40 {
		client.Say(msg.Channel, "Nah. Maybe some other time.")
	} else {
		client.Say(msg.Channel, fmt.Sprintf("Check out %s on their twitch channel. I'm sure you can find it yourself.", args[1]))
		client.Say(msg.Channel, "If they even have one... I'm too lazy to validate it.")
	}
}

func (s *Shoutout) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {

}
