package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gempir/go-twitch-irc/v2"
)

type Ded struct{}
type counter struct {
	Count int
}

var cooldown int = 15
var locked = false

func (d *Ded) Init() {

}

func (d *Ded) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	if locked && !IsMod(msg.User) {
		return
	}
	// Mods can run this during cooldown - but don't elongate the cooldown
	if !locked {
		locked = true
		go unlock()
	}
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if len(args) > 2 {
		client.Say(msg.Channel, "Too many arguments to ded. Why you do dis?")
		return
	}
	count := -1
	var err error
	if len(args) == 2 {
		if !IsMod(msg.User) {
			client.Say(msg.Channel, "Only mods can set the counter directly.")
			return
		}
		count, err = strconv.Atoi(args[1])
		if err != nil {
			client.Say(msg.Channel, "ded requires a number not a thing else")
			return
		}
	}

	u := "http://localhost:8080/inc_count"
	if count >= 0 {
		u = "http://localhost:8080/set_count"
	}
	resp, err := http.PostForm(u, url.Values{"count": {strconv.Itoa(count)}})
	if err != nil {
		client.Say(msg.Channel, "ded counter seems to be off")
		log.Println(err.Error())
		return
	}
	c := counter{}
	err = json.NewDecoder(resp.Body).Decode(&c)
	if err != nil {
		client.Say(msg.Channel, "I'm sorry, I messed up. Try again some other decade.")
		log.Println(err.Error())
		return
	}
	if c.Count == 0 {
		client.Say(msg.Channel, "ded counter reset")
		return
	}
	var plural string
	if c.Count > 1 {
		plural = "s"
	}
	client.Say(msg.Channel, fmt.Sprintf("Has ded %d time%s.", c.Count, plural))
	if c.Count == 1 {
		time.Sleep(time.Millisecond * time.Duration(1000))
		client.Say(msg.Channel, "ONE TIME!")
	}
}

func (d *Ded) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {

}

func unlock() {
	time.Sleep(time.Second * time.Duration(cooldown))
	locked = false
}

func (d *Ded) Help() []string {
	return []string{
		"!ded to increment the ded counter because streamer is bad at game",
	}
}
