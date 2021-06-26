package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type Lights struct{}

var lightLock bool = false
var lightCD int = 5
var bridgeID string = os.Getenv("HUE_BRIDGE_ID")
var lights *Lights = &Lights{}

const (
	red   int = 0
	green int = 25500
	blue  int = 46920
)

func init() {
	RegisterCommand("lights", lights)
}

func NewLights() *Lights {
	return &Lights{}
}

func (l *Lights) PostInit() {

}

func (l *Lights) Run(msg twitch.PrivateMessage) {
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
			comm.ToChat(msg.Channel, "I only know red, green and blue right now...")
			lightLock = false
			return
		}
	}
	if color > 65535 || color < 0 {
		comm.ToChat(msg.Channel, "Invalid color value soz")
		lightLock = false
		return
	}

	comm.ToOverlay(fmt.Sprintf("lights set %d", color))

	go func() {
		time.Sleep(time.Second * time.Duration(lightCD))
		lightLock = false
	}()
}

func (l *Lights) Help() []string {
	return []string{
		"!lights [color] to set the color of the lights you can't see.",
		"red, green, blue or an integer value from 0 to 65535",
	}
}
