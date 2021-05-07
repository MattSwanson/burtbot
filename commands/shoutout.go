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
	//customMessages map[string]string // key is username, value is a message to display
}

func (s *Shoutout) Init() {
	rand.Seed(time.Now().UnixNano())
}

func (s *Shoutout) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	if !isMod(msg.User) {
		return
	}
	if !s.TwitchClient.GetAuthStatus() {
		client.Say(msg.Channel, "I'd shout them out or whatever but I don't have \"ACCESS\" to the info... hint hint.")
		return
	}
	args := strings.Fields(strings.ToLower(msg.Message))
	if len(args) < 2 {
		return
	}
	u := s.TwitchClient.GetUser(args[1])
	if u.UserID == "" {
		client.Say(msg.Channel, "Sorry, I don't shout out non-existant users. Not for free at least.")
		return
	}
	ci := s.TwitchClient.GetChannelInfo(u.UserID)
	var game string
	if ci.GameName == "" {
		game = "<REDACTED>"
	} else {
		game = ci.GameName
	}
	r := rand.Intn(100)
	if len(args) == 3 {
		if args[2] == "please" || args[2] == "plz" {
			client.Say(msg.Channel, "Fine...")
			client.Say(msg.Channel, fmt.Sprintf("Check out %s on their twitch channel: http://twitch.tv/%[1]s", u.DisplayName))
			client.Say(msg.Channel, fmt.Sprintf("They were last seen streaming %s. Whatever that is.", game))
			return
		}
	}
	if r < 80 {
		client.Say(msg.Channel, "Nah. Maybe some other time.")
	} else {
		client.Say(msg.Channel, fmt.Sprintf("CHECK OUT %s ON AT http://twitch.tv/%[1]s", u.DisplayName))
		client.Say(msg.Channel, fmt.Sprintf("THEY WERE LAST SEEN STREAMING %s. WHATEVER THAT IS.", game))
	}
}

func (s *Shoutout) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {

}

func (s *Shoutout) Help() []string {
	return []string{
		"!so [user] to shout out another streamer",
		"Sometimes you have to ask nicely",
	}
}
