package commands

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gempir/go-twitch-irc/v2"
)

type Lights struct{}

var lightLock bool = false
var lightCD int = 5

var bridgeID string = os.Getenv("HUE_BRIDGE_ID")

const (
	red   int = 0
	green int = 25500
	blue  int = 46920
)

func (l *Lights) Init() {

}

func (l *Lights) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	// !lights red or !lights 5500
	if lightLock {
		return
	}
	lightLock = true
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if len(args) < 2 {
		lightLock = false
		return
	}

	// check to see if the input could be an int...
	color, err := strconv.Atoi(args[1])
	if err != nil {
		// if we got a string that's not a number
		// just three colors for now...
		switch strings.ToLower(args[1]) {
		case "red":
			color = red
		case "green":
			color = green
		case "blue":
			color = blue
		default:
			client.Say(msg.Channel, "I only know red, green and blue right now...")
			lightLock = false
			return
		}
	}
	if color > 65535 || color < 0 {
		client.Say(msg.Channel, "Invalid color value soz")
		lightLock = false
		return
	}

	endPoint := fmt.Sprintf("http://10.0.0.2/api/%s/groups/1/action", bridgeID)
	reqBody := fmt.Sprintf(`{"on":true, "hue":%d}`, color)
	br := strings.NewReader(reqBody)
	req, err := http.NewRequest("PUT", endPoint, br)
	if err != nil {
		client.Say(msg.Channel, "Lights needs to be refilled")
		log.Println(err.Error())
		lightLock = false
		return
	}
	_, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err.Error())
		lightLock = false
		return
	}
	go func() {
		time.Sleep(time.Second * time.Duration(lightCD))
		lightLock = false
	}()
}

func (l *Lights) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {
	return
}
