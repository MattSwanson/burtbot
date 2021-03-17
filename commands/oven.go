package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"

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
			client.Say(msg.Channel, "Use !oven preheat <temperature>")
			return
		}
		temp, err := strconv.Atoi(args[2])
		if err != nil {
			client.Say(msg.Channel, "The oven can only heat up to temperatures specified by numbers...")
			return
		}
		client.Say(msg.Channel, fmt.Sprintf("Preheating the oven to %d degrees. This may take a while...", temp))
		o.BakeTemp = temp
		go func() {
			o.Preheat(temp)
			client.Say(msg.Channel, fmt.Sprintf("Ding! Oven is heated to %d", temp))
		}()
	case "temp":
		client.Say(msg.Channel, fmt.Sprintf("Oven is at %df", o.Temperature))
	}
}

func (o *Oven) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {
	return
}

func (o *Oven) Preheat(temp int) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			o.Temperature = o.Temperature + 1
			if o.Temperature >= temp {
				o.Temperature = temp
				return
			}
		}
	}
}
