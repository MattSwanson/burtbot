package commands

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type Cube struct{}

var cube *Cube = &Cube{}
var cubeRunning bool
var cubeStartingState []byte

var validMoveChars []byte = []byte{
	'R', 'r', 'L', 'l', 'U', 'u', 'D', 'd', 'B', 'b', 'F', 'f',
	'\'', 'X', 'x', 'Y', 'y', 'Z', 'z', 'M', 'm', '2', 'E', 'S',
}

func init() {
	comm.SubscribeToReply("cube", cube.SaveCube)
	RegisterCommand("cube", cube)
	loadStateFromFile()
}

func loadStateFromFile() {
	var err error
	cubeStartingState, err = os.ReadFile("cube.json")
	if err != nil {
		log.Println("Oh noes, couldn't load cube state from file :( ", err)
	}
}

func (c *Cube) Run(msg twitch.PrivateMessage) {
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if len(args) < 2 {
		return
	}

	switch args[1] {
	case "move":
		if len(args) < 3 {
			return
		}
		if !validateMoves(args[2]) {
			comm.ToChat(msg.Channel, "Invalid moves")
			return
		}
		comm.ToOverlay(fmt.Sprintf("cube move %s", args[2]))
	case "start":
		loadStateFromFile()
		comm.ToOverlay(fmt.Sprintf("cube start %s", string(cubeStartingState)))
	case "stop":
		comm.ToOverlay("cube stop")
	case "reset":
		comm.ToOverlay("cube reset")
	case "shuffle":
		comm.ToOverlay("cube shuffle")
	case "movecount":
		comm.ToOverlay("cube movecount")
	case "pos":
		if len(args) < 5 || !IsMod(msg.User) {
			return
		}
		comm.ToOverlay(fmt.Sprintf("cube pos %s %s %s", args[2], args[3], args[4]))
	}
}

func (c *Cube) PostInit() {

}

func (c *Cube) Help() []string {
	return []string{
		"!cube move [move] to manipulate the cube",
		"See https://ruwix.com/the-rubiks-cube/notation/ for move notation",
	}
}

func (c *Cube) SaveCube(args []string) {
	if len(args) < 2 {
		return
	}
	if err := os.WriteFile("cube.json", []byte(args[1]), 0644); err != nil {
		log.Println("Error saving cube state to file: ", err)
	}
}

func validateMoves(moves string) bool {
	for i := 0; i < len(moves); i++ {
		// is ch in the valid move slice?
		var found bool
		for j := 0; j < len(validMoveChars); j++ {
			if moves[i] == validMoveChars[j] {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
