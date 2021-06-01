//TODO:
// Commands should have their own init section for setup to keep things as modular as possible
// Commands should also have fields for cooldowns, help, etc...

package commands

import (
	"errors"
	"fmt"
	"strings"
	"os"
	"log"
	"encoding/json"

	"github.com/gempir/go-twitch-irc/v2"
)

const (
	aliasesFileName = "./aliases.json"
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
	aliases	   map[string]string
	TcpChannel chan string
}

// type Command struct {
// 	Run func(twitch.PrivateMessage)
// }

func NewCmdHandler(client *twitch.Client, tcpChannel chan string) *CmdHandler {
	return &CmdHandler{
		Client:     client,
		Commands:   make(map[string]Command),
		aliases:	make(map[string]string),
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

func IsMod(user twitch.User) bool {
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

func (handler *CmdHandler) LoadAliases() {
	j, err := os.ReadFile(aliasesFileName)
	if err != nil {
		log.Println("couldn't load aliases from file")
		return
	}
	err = json.Unmarshal(j, &handler.aliases)
	if err != nil {
		log.Println("invalid json in aliases file")
	}
	for alias, commandName := range handler.aliases {
		if err := handler.RegisterAlias(alias, commandName); err != nil {
			log.Println("couldn't register alias from file")
		}
	}
}

func (handler *CmdHandler) RegisterAlias(alias, commandName string) error {
	// check to see if the command exists in the commands map
	cmd, ok := handler.Commands[commandName]
	if !ok { 
		return errors.New("command doesn't exist, can not assign alias")
	}
	handler.Commands[alias] = cmd
	handler.aliases[alias] = commandName
	handler.saveAliasesToFile()
	return nil
}

func (handler *CmdHandler) RemoveAlias(alias string) {
	if _, ok := handler.Commands[alias]; ok {
		delete(handler.Commands, alias)
		delete(handler.aliases, alias)
		handler.saveAliasesToFile()
	}
}

func (handler *CmdHandler) saveAliasesToFile() {
	json, err := json.Marshal(handler.aliases)
	if err != nil {
		log.Println("couldn't convert alias map to json")
		return
	}
	if err := os.WriteFile(aliasesFileName, json, 0644); err != nil {
		log.Println(err.Error())
	}
}
