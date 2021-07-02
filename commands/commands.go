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
	"net/http"
	"html/template"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

const (
	aliasesFileName = "./aliases.json"
)

var cmdHandler *CmdHandler = &CmdHandler{Commands: make(map[string]Command)}
var onPartSubscriptions []func(twitch.UserPartMessage)
var onJoinSubscriptions []func(twitch.UserJoinMessage)
var rawMsgSubscriptions []func(twitch.PrivateMessage)
var helpTemplate *template.Template

type Command interface {
	Run(twitch.PrivateMessage)
	PostInit()
	Help() []string
}

type CmdHandler struct {
	Client     *twitch.Client
	Commands   map[string]Command
	aliases	   map[string]string
}

type cmdHelp struct {
	Name string
	Help []string
}

// type Command struct {
// 	Run func(twitch.PrivateMessage)
// }

func init() {
	http.HandleFunc("/commands", commandList) 
	helpTemplate = template.Must(template.ParseFiles("templates/help.gohtml"))
}

func NewCmdHandler(client *twitch.Client) *CmdHandler {
	cmdHandler.Client = client
	cmdHandler.aliases = make(map[string]string)
	return cmdHandler
}

func (handler *CmdHandler) RegisterCommand(pattern string, c Command) error {
	if _, ok := handler.Commands[pattern]; ok {
		return errors.New("Command already registered with that pattern")
	}
	// c.Init()
	handler.Commands[pattern] = c
	return nil
}

func RegisterCommand(pattern string, c Command) error {
	return cmdHandler.RegisterCommand(pattern, c)
}

func (handler *CmdHandler) PostInit() {
	for _, c := range handler.Commands {
		c.PostInit()		
	}
}

func (handler *CmdHandler) HandleMsg(msg twitch.PrivateMessage) {
	for _, fn := range rawMsgSubscriptions {
		fn(msg)
	}
	if msg.Message == "w" {
		comm.ToOverlay("up")
	}
	if msg.Message == "a" {
		comm.ToOverlay("left")
	}
	if msg.Message == "s" {
		comm.ToOverlay("down")
	}
	if msg.Message == "d" {
		comm.ToOverlay("right")
	}

	if !strings.HasPrefix(msg.Message, "!") {
		return
	}
	msg.Message = handler.InjectAliases(msg.Message)
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	lower := strings.ToLower(msg.Message)
	if lower == "!help" {
		handler.HelpAll()
	}
	fields := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if IsMod(msg.User) && fields[0] == "alias" {
		// !alias add alias command
		if len(fields) > 3 && fields[1] == "add" {
			originalCommand := strings.Join(fields[3:], " ")
			err := handler.RegisterAlias(fields[2], originalCommand)			
			if err != nil {
				comm.ToChat(msg.Channel, fmt.Sprintf("The alias [%s] already exists.", fields[2]))
				return
			}
			comm.ToChat(msg.Channel, fmt.Sprintf("Created alias [%s] for [%s]", fields[2], originalCommand)) 
			return
		}
		if len(fields) > 2 && fields[1] == "remove" {
			if handler.RemoveAlias(fields[2]) {
				comm.ToChat(msg.Channel, fmt.Sprintf("Successfully removed alias [%s]", fields[2]))
			}
		}
	}

	if lower == "!commands" {
		comm.ToChat(msg.Channel, "See available commands at: https://burtbot.app/commands")
		return
	}

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
		go cmd.Run(msg)
	}
}

func (handler *CmdHandler) HandlePartMsg(msg twitch.UserPartMessage) {
	// notify any commands that require it - that a user has parted the channel
	for _, fn := range onPartSubscriptions {
		fn(msg)
	}
}

func (handler *CmdHandler) HandleJoinMsg(msg twitch.UserJoinMessage) {
	for _, fn := range onJoinSubscriptions {
		fn(msg)
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
			comm.ToOverlay(fmt.Sprintf("tts true %s", h))
			comm.ToOverlay(fmt.Sprintf("marquee once {\"rawMessage\":\"%s\"}", h))
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
}

func (handler *CmdHandler) RegisterAlias(alias, commandName string) error {
	fmt.Println(commandName)
	if _, ok := handler.aliases[alias]; ok {
		return errors.New("alias already exists")
	}
	handler.aliases[alias] = commandName
	handler.saveAliasesToFile()
	return nil
}

func (handler *CmdHandler) RemoveAlias(alias string) bool {
	fmt.Println("trying to remove ", alias)
	if _, ok := handler.aliases[alias]; !ok {
		return false
	}
	delete(handler.Commands, alias)
	delete(handler.aliases, alias)
	handler.saveAliasesToFile()
	return true
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

func (handler *CmdHandler) InjectAliases(message string) string {
	// check to see if the command entered is an alias
	fields := strings.Fields(strings.TrimPrefix(message, "!"))
	command, ok := handler.aliases[fields[0]] 
	if !ok {
		return message
	}
	// if so replace the alias with the command it represents
	fields[0] = "!" + command
	return strings.Join(fields, " ")
}

func GetCommandMap() *map[string]Command {
	return &cmdHandler.Commands
}

func SubscribeUserPart(f func(twitch.UserPartMessage)) {
	onPartSubscriptions = append(onPartSubscriptions, f)
}

func SubscribeUserJoin(f func(twitch.UserJoinMessage)) {
	onJoinSubscriptions = append(onJoinSubscriptions, f)
}

func SubscribeToRawMsg(f func(twitch.PrivateMessage)) {
	rawMsgSubscriptions = append(rawMsgSubscriptions, f)
}

// show a list of commands and their arguments
func commandList(w http.ResponseWriter, r *http.Request) {
	cmds := []cmdHelp{}
	for cmdName, cmd := range cmdHandler.Commands {
		c := cmdHelp{
			Name: cmdName,
			Help: []string{},
		}
		for _, h := range cmd.Help() {
			c.Help = append(c.Help, h)	
		}
		cmds = append(cmds, c)
	}
	err := helpTemplate.ExecuteTemplate(w, "help.gohtml", cmds)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
