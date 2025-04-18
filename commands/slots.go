package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type Slots struct {
	isRunning   bool
	currentUser string
}

var slots *Slots = &Slots{}

func init() {
	comm.SubscribeToReply("slots", slots.HandleReply)
	RegisterCommand("slots", slots)
}

func (s *Slots) Run(msg twitch.PrivateMessage) {
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if len(args) < 2 {
		return
	}

	switch args[1] {
	case "kick":
		comm.ToOverlay("slots kick")
	case "pull":
		if s.isRunning {
			comm.ToChat(msg.Channel, fmt.Sprintf("Someone is already using the slots, please wait your turn patiently and don't complain about anything. If you really need to complain, direct it at @%s since they are the one who is holding everything up.", s.currentUser))
			return
		}
		if len(args) < 3 {
			comm.ToChat(msg.Channel, "You must give a bet amount to pull the slots")
			return
		}
		val, err := strconv.Atoi(args[2])
		if err != nil || val <= 0 {
			comm.ToChat(msg.Channel, "Bet amount is invalid, please try again with a number.")
			return
		}
		s.isRunning = true
		s.currentUser = msg.User.DisplayName
		comm.ToOverlay(fmt.Sprintf("slots pull %s %s", args[2], msg.User.DisplayName))
	case "start":
		comm.ToOverlay("slots start")
	case "stop":
		comm.ToOverlay("slots stop")
	}
}

func (s *Slots) PostInit() {

}

func (s *Slots) HandleReply(args []string) {
	// slots result user payout
	if len(args) < 4 {
		log.Println("Got invalid slots result from overlay: ", strings.Join(args, " "))
		return
	}
	payout, err := strconv.Atoi(args[3])
	if err != nil {
		log.Println("Invalid payout from slots: ", args[3])
		return
	}
	GrantToken(strings.ToLower(args[2]), big.NewInt(int64(payout)))

	plural := ""
	if payout > 1 {
		plural = "s"
	}

	msg := fmt.Sprintf("@%s won %d token%s from the slot machine!", args[2], payout, plural)
	if payout <= 0 {
		msg = fmt.Sprintf("@%s was SCAMMED by the slot machine. Tough luck!", args[2])
	}

	mMsg := MarqueeMsg{
		RawMessage: msg,
		Emotes:     "",
	}
	json, err := json.Marshal(mMsg)
	if err != nil {
		return
	}
	comm.ToOverlay("marquee once " + string(json))
	s.currentUser = ""
	s.isRunning = false
}

func (s *Slots) Help() []string {
	return []string{}
}
