package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type Oven struct {
	Temperature int
	Preheated   bool
	BakeTemp    int
	Contents    food
}

type food struct {
	Name        string
	BakeTemp    int
	Coductivity float64 // how quickly the food will cook
}

func (o *Oven) Init() {

}

func (o *Oven) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if len(args) < 2 {
		return
	}
	switch args[1] {
	case "preheat":
		if len(args) != 3 {
			comm.ToChat(msg.Channel, "Use !oven preheat <temperature>")
			return
		}
		temp, err := strconv.Atoi(args[2])
		if err != nil {
			comm.ToChat(msg.Channel, "The oven can only heat up to temperatures specified by numbers...")
			return
		}
		comm.ToChat(msg.Channel, fmt.Sprintf("Preheating the oven to %d degrees. This may take a while...", temp))
		o.BakeTemp = temp
		go func() {
			o.Preheat(temp)
			comm.ToChat(msg.Channel, fmt.Sprintf("Ding! Oven is heated to %d", temp))
		}()
	case "temp":
		comm.ToChat(msg.Channel, fmt.Sprintf("Oven is at %d degrees", o.Temperature))
	}
}

func (o *Oven) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {

}

func (o *Oven) Preheat(temp int) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		o.Temperature = o.Temperature + 1
		if o.Temperature >= temp {
			o.Temperature = temp
			return
		}
	}
}

func (o *Oven) Help() []string {
	return []string{
		"!oven preheat [temp] to preheat the oven to the specified temperature",
		"!oven temp to check the current temperature of the oven",
		"Wonder why... Why anything?",
	}
}
