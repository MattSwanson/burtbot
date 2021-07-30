package commands

import (
	"bufio"
	"io"
	"log"
	"os/exec"
	"strings"

	"github.com/MattSwanson/burtbot/console"
	"github.com/gempir/go-twitch-irc/v2"
)

type Cowsay struct{}

var cs *Cowsay = &Cowsay{}

func init() {
	RegisterCommand("cowsay", cs)
}

func (cs *Cowsay) Run(msg twitch.PrivateMessage) {
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if len(args) < 2 || args[1] == "help" {
		return
	}
	str := strings.Join(args[1:], " ")
	if strings.HasPrefix(str, "-") {
		return
	}
	cmd := exec.Command("cowsay", str)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println(err)
		return
	}
	if err := cmd.Start(); err != nil {
		log.Println(err)
		return
	}
	reader := bufio.NewReader(stdout)
	for {
		ln, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println(err)
			return
		}
		ln = ln[:len(ln)-1] // trim newline char from end
		console.AddMessage(ln, console.Green)
	}
	if err := cmd.Wait(); err != nil {
		log.Println(err)
	}
}

func (cs *Cowsay) PostInit() {

}

func (cs *Cowsay) Help() []string {
	return []string{
		"!cowsay [some text here] to make the cow... say the thing.",
	}
}
