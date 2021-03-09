package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gempir/go-twitch-irc/v2"
)

// Bbset will allow text commands to be saved and used later
//TODO Db implementation
type Bbset struct {
	Commands map[string]string
	// This is a hack to not allow chat commands to be named the same as bot functions
	// Probably a better way to do this in the command handler it self but this is
	// where it is for now
	ReservedCommands *map[string]Command
	Persist          bool
}

// Run will be used to set commands, then commands will be run from a different method
func (b Bbset) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	//fmt.Println("")
	if !isMod(msg.User) {
		return
	}
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	// args[0] we know is bbset
	// args[1] will be the name of the command
	// anything beyond will be the text to save
	if len(args) < 2 {
		client.Say(msg.Channel, "Please provide a command to create")
		return
	}
	if _, ok := (*b.ReservedCommands)[args[1]]; ok {
		client.Say(msg.Channel, "There is already a bot function with that name")
		return
	}
	if len(args) < 3 {
		client.Say(msg.Channel, "Nothing provided to say...")
		return
	}
	_, ok := b.Commands[args[1]]
	if ok {
		// Is this marked for removal
		if args[2] == "remove" {
			delete(b.Commands, args[1])
			client.Say(msg.Channel, fmt.Sprintf("Removed command %s", args[1]))
			if b.Persist {
				b.saveCommandsToFile()
			}
			return
		}
		client.Say(msg.Channel, "There is already a command with that name")
		return
	}
	b.Commands[args[1]] = strings.Join(args[2:], " ")
	if b.Persist {
		b.saveCommandsToFile()
	}
}

func (b Bbset) HandleMsg(client *twitch.Client, msg twitch.PrivateMessage) {
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if len(args) > 1 {
		return
	}
	if txt, ok := b.Commands[args[0]]; ok {
		client.Say(msg.Channel, txt)
	}
}

func (b Bbset) saveCommandsToFile() {
	json, err := json.Marshal(b.Commands)
	if err != nil {
		log.Println("Couldn't json")
		return
	}
	if err := os.WriteFile("./commands.json", json, 0644); err != nil {
		log.Println(err.Error())
	}
}
