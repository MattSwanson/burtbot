package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

// Bbset will allow text commands to be saved and used later
//TODO Db implementation
type Bbset struct {
	// This is a hack to not allow chat commands to be named the same as bot functions
	// Probably a better way to do this in the command handler it self but this is
	// where it is for now
	ReservedCommands *map[string]Command

	commands map[string]string
	persist  bool
}

func (b *Bbset) Init() {
	b.commands = make(map[string]string)
	b.persist = true
	j, err := os.ReadFile("./commands.json")
	if err != nil {
		log.Println("Couldn't loat chat commands from file")
		b.persist = false
	}
	err = json.Unmarshal(j, &b.commands)
	if err != nil {
		log.Println("Invalid json in chat commands file")
		b.persist = false
	}
}

// Run will be used to set commands, then commands will be run from a different method
func (b *Bbset) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	//fmt.Println("")
	if !IsMod(msg.User) {
		return
	}
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	// args[0] we know is bbset
	// args[1] will be the name of the command
	// anything beyond will be the text to save
	if len(args) < 2 {
		comm.ToChat(msg.Channel, "Please provide a command to create")
		return
	}
	if _, ok := (*b.ReservedCommands)[args[1]]; ok {
		comm.ToChat(msg.Channel, "There is already a bot function with that name")
		return
	}
	if len(args) < 3 {
		comm.ToChat(msg.Channel, "Nothing provided to say...")
		return
	}
	_, ok := b.commands[args[1]]
	if ok {
		// Is this marked for removal
		if args[2] == "remove" {
			delete(b.commands, args[1])
			comm.ToChat(msg.Channel, fmt.Sprintf("Removed command %s", args[1]))
			if b.persist {
				b.saveCommandsToFile()
			}
			return
		}
		comm.ToChat(msg.Channel, "There is already a command with that name")
		return
	}
	b.commands[args[1]] = strings.Join(args[2:], " ")
	comm.ToChat(msg.Channel, fmt.Sprintf("!%s set for: %s", args[1], b.commands[args[1]]))
	if b.persist {
		b.saveCommandsToFile()
	}
}

func (b *Bbset) HandleMsg(client *twitch.Client, msg twitch.PrivateMessage) {
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if len(args) > 1 || len(args) == 0 {
		return
	}
	if txt, ok := b.commands[args[0]]; ok {
		comm.ToChat(msg.Channel, txt)
	}
}

func (b *Bbset) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {

}

func (b *Bbset) saveCommandsToFile() {
	json, err := json.Marshal(b.commands)
	if err != nil {
		log.Println("Couldn't json")
		return
	}
	if err := os.WriteFile("./commands.json", json, 0644); err != nil {
		log.Println(err.Error())
	}
}

func (b *Bbset) Help() []string {
	return []string{
		"Create a text based command",
		"To add - !bbset [name] [text to display]",
		"To remove - !bbset [name] remove",
		"Then use ![name] to run the command",
	}
}
