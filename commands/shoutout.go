package commands

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/MattSwanson/burtbot/db"
	"github.com/MattSwanson/burtbot/helix"
	"github.com/gempir/go-twitch-irc/v2"
)

type Shoutout struct {
	//customMessages map[string]string // key is username, value is a message to display
}

var shoutOut *Shoutout = &Shoutout{}
var autoShouts []*autoShout

type autoShout struct {
	user db.User
	shouted bool
}

func init() {
	SubscribeToRawMsg(checkAutoShout)
	RegisterCommand("so", shoutOut)
}

func (s *Shoutout) PostInit() {
	rand.Seed(time.Now().UnixNano())
	autoShouts = []*autoShout{}
	users, err := db.GetAutoShoutUsers()
	if err != nil {
		log.Println("couldn't get autoshout users from DB")
	}
	for _, u := range users {
		autoShouts = append(autoShouts, &autoShout{u, false})
	}
}

func (s *Shoutout) Run(msg twitch.PrivateMessage) {
	if !IsMod(msg.User) {
		return
	}
	args := strings.Fields(strings.ToLower(msg.Message))
	if len(args) == 2 {
		shout(args[1], msg.Channel)
		return
	}
	if len(args) == 3 && args[1] == "as" && IsMod(msg.User) {
		// add to auto shouts
		addToAutoShout(args[2], msg.Channel)
		return
	}
	if len(args) == 3 && args[1] == "remove" && IsMod(msg.User) {
		// add to auto shouts
		removeAutoShout(args[2], msg.Channel)
	}
}

func checkAutoShout(msg twitch.PrivateMessage) {
	for _, as := range autoShouts {
		userID, _ := strconv.Atoi(msg.User.ID)
		if userID == as.user.TwitchID && !as.shouted {
			shout(as.user.DisplayName, msg.Channel)
			as.shouted = true
		}
	}
}

func shout(username, channel string) {
	if !helix.GetAuthStatus() {
		comm.ToChat(channel, "I'd shout them out or whatever but I don't have \"ACCESS\" to the info... hint hint.")
		return
	}
	u := helix.GetUser(username)
	if u.UserID == "" {
		comm.ToChat(channel, "Sorry, I don't shout out non-existant users. Not for free at least.")
		return
	}
	ci := helix.GetChannelInfo(u.UserID)
	var game string
	if ci.GameName == "" {
		game = "<REDACTED>"
	} else {
		game = ci.GameName
	}
	/*r := rand.Intn(100)
	if len(args) == 3 {
		if args[2] == "please" || args[2] == "plz" {
			comm.ToChat(msg.Channel, "Fine...")
			comm.ToChat(msg.Channel, fmt.Sprintf("Check out %s on their twitch channel: http://twitch.tv/%[1]s", u.DisplayName))
			comm.ToChat(msg.Channel, fmt.Sprintf("They were last seen streaming %s. Whatever that is.", game))
			return
		}
	}
	if r < 80 {
		comm.ToChat(msg.Channel, "Nah. Maybe some other time.")
	} else {
		comm.ToChat(msg.Channel, fmt.Sprintf("CHECK OUT %s ON AT http://twitch.tv/%[1]s", u.DisplayName))
		comm.ToChat(msg.Channel, fmt.Sprintf("THEY WERE LAST SEEN STREAMING %s. WHATEVER THAT IS.", game))
	}*/
	comm.ToChat(channel, fmt.Sprintf("CHECK OUT %s ON AT http://twitch.tv/%[1]s", u.DisplayName))
	comm.ToChat(channel, fmt.Sprintf("THEY WERE LAST SEEN STREAMING %s. WHATEVER THAT IS.", game))
}

func removeAutoShout(username, channel string) {
	found := false
	for k, as := range autoShouts {
		if as.user.DisplayName == username {
			as.user.AutoShout = false
			db.AddUser(as.user)
			autoShouts[k], autoShouts[len(autoShouts)-1] = autoShouts[len(autoShouts)-1], autoShouts[k]
			autoShouts = autoShouts[:len(autoShouts)-1]
			found = true
		}
	}
	if !found {
		comm.ToChat(channel, fmt.Sprintf("%s was not in the auto shout list", username))
		return
	}
	comm.ToChat(channel, fmt.Sprintf("Removed %s from the auto shout list", username))
}

func addToAutoShout(username, channel string) {
	if !helix.GetAuthStatus() {
		comm.ToChat(channel, "Can't add user to auto shout since I'm not authed for Twitch...")
		return
	}
	user := helix.GetUser(username)
	if user.UserID == "" {
		comm.ToChat(channel, fmt.Sprintf("Couldn't get infor for %s from Twitch.", username))
		return
	}
	intID, err := strconv.Atoi(user.UserID)
	if err != nil {
		comm.ToChat(channel, "I ran into an issue updating the user database...")
		return
	}
	u := db.User{
		TwitchID: intID,
		DisplayName: username,
		AutoShout: true,
	}
	db.AddUser(u)
	if err != nil {
		comm.ToChat(channel, "I ran into an issue updating the user database...")
		return
	}
	autoShouts = append(autoShouts, &autoShout{u, false})
	comm.ToChat(channel, fmt.Sprintf("Added %s to the auto shout list", username))
}

func (s *Shoutout) Help() []string {
	return []string{
		"!so [user] to shout out another streamer",
		"Sometimes you have to ask nicely",
	}
}
