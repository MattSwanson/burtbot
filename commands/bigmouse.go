package commands

import (
	"fmt"
	"time"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type BigMouse struct {
	lastUse time.Time
}

const (
	bigMouseCooldown = 600 // seconds
	duration         = 60  // seconds
)

var (
	bigMouse *BigMouse = &BigMouse{}
)

func init() {
	RegisterCommand("bigmouse", bigMouse)
}

func (m *BigMouse) PostInit() {

}

func (m *BigMouse) Run(msg twitch.PrivateMessage) {
	if timeSince := time.Since(m.lastUse).Seconds(); timeSince <= float64(bigMouseCooldown) {
		comm.ToChat(msg.Channel, fmt.Sprintf("Bigmouse is on cooldown for another %d seconds. Patience...", bigMouseCooldown-int(timeSince)))
		return
	}
	m.lastUse = time.Now()
	comm.ToOverlay(fmt.Sprintf("bigmouse %d", duration))
}

func (m *BigMouse) Help() []string {
	return []string{
		"!bigmouse to toggle big mouse mode [NOT WORKING RIGHT NOW... lazy]",
	}
}
