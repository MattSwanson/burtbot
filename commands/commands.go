//TODO:
// Commands should have their own init section for setup to keep things as modular as possible
// Commands should also have fields for cooldowns, help, etc...

package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gempir/go-twitch-irc/v2"
)

type Command interface {
	Run(*twitch.Client, twitch.PrivateMessage)
	Init()
	OnUserPart(*twitch.Client, twitch.UserPartMessage)
	Help() []string
}

type CmdHandler struct {
	Client     *twitch.Client
	Commands   map[string]Command
	TcpChannel chan string
}

// type Command struct {
// 	Run func(twitch.PrivateMessage)
// }

func NewCmdHandler(client *twitch.Client, tcpChannel chan string) *CmdHandler {
	return &CmdHandler{
		Client:     client,
		Commands:   make(map[string]Command),
		TcpChannel: tcpChannel,
	}
}

func (handler *CmdHandler) RegisterCommand(pattern string, c Command) error {
	if _, ok := handler.Commands[pattern]; ok {
		return errors.New("Command already registered with that pattern")
	}
	handler.Commands[pattern] = c
	return nil
}

func (handler *CmdHandler) HandleMsg(msg twitch.PrivateMessage) {
	if !strings.HasPrefix(msg.Message, "!") {
		return
	}
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if len(args) == 0 {
		return
	}
	lcmd := strings.ToLower(args[0])
	if cmd, ok := handler.Commands[lcmd]; ok {
		if len(args) > 1 && args[1] == "help" {
			for _, h := range cmd.Help() {
				handler.Client.Say(msg.Channel, h)
			}
		}
		go cmd.Run(handler.Client, msg)
	}
}

func (handler *CmdHandler) HandlePartMsg(msg twitch.UserPartMessage) {
	// notify any commands that require it - that a user has parted the channel
	for _, command := range handler.Commands {
		command.OnUserPart(handler.Client, msg)
	}
}

func isMod(user twitch.User) bool {
	_, bcOk := user.Badges["broadcaster"]
	_, modOk := user.Badges["moderator"]
	return bcOk || modOk
}

// Show all the commands help text.. all of them... at once.
// Or say them all????
func (handler *CmdHandler) HelpAll() {
	for _, cmd := range handler.Commands {
		for _, h := range cmd.Help() {
			handler.TcpChannel <- fmt.Sprintf("tts true %s", h)
			handler.TcpChannel <- fmt.Sprintf("marquee once {\"rawMessage\":\"%s\"}", h)
		}
	}
}
