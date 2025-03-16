package commands

import (
	"fmt"
	"time"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type Flashlight struct {
	lastUse time.Time
}

const (
	flashlightCooldown = 600 // seconds
	flashlightDuration = 60  // seconds
)

var (
	flashlight *Flashlight = &Flashlight{}
)

func init() {
	RegisterCommand("flashlight", flashlight)
}

func (f *Flashlight) PostInit() {

}

func (f *Flashlight) Run(msg twitch.PrivateMessage) {
	if timeSince := time.Since(f.lastUse).Seconds(); timeSince <= float64(flashlightCooldown) {
		comm.ToChat(msg.Channel, fmt.Sprintf("Flashlight is on cooldown for another %d seconds.", flashlightCooldown-int(timeSince)))
		return
	}
	f.lastUse = time.Now()
	comm.ToOverlay(fmt.Sprintf("flashlight %d", flashlightDuration))
}

func (f *Flashlight) Help() []string {
	return []string{
		"!flashlight to turn off the lights on the streamer. (10 minute cooldown)",
	}
}
