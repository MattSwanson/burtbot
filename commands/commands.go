package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/MattSwanson/burtbot/console"
	"github.com/MattSwanson/burtbot/helix"
	"github.com/gempir/go-twitch-irc/v2"
)

const (
	aliasesFileName = "./aliases.json"
	helpAllCooldown = 300 // seconds
)

var cmdHandler *CmdHandler = &CmdHandler{Commands: make(map[string]Command)}
var onPartSubscriptions []func(twitch.UserPartMessage)
var onJoinSubscriptions []func(twitch.UserJoinMessage)
var rawMsgSubscriptions []func(twitch.PrivateMessage)
var helpTemplate *template.Template
var lastHelpAll time.Time
var mobileStream bool

type Command interface {
	Run(twitch.PrivateMessage)
	PostInit()
	Help() []string
}

type CmdHandler struct {
	Client   *twitch.Client
	Commands map[string]Command
	aliases  map[string]string
}

type cmdHelp struct {
	Name    string
	Help    []string
	Aliases map[string]string
}

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
	handler.Commands[pattern] = c
	return nil
}

func (handler *CmdHandler) PostInit() {
	helix.SubscribeToFollowEvent(FollowAlertToOverlay)
	helix.SubscribeToRaidEvent(RaidAlertToOverlay)
	for _, c := range handler.Commands {
		c.PostInit()
	}
}

func (handler *CmdHandler) HandleMsg(msg twitch.PrivateMessage) {
	if msg.Message == "!" {
		return
	}
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
	comm.ToOverlay("lights set 0")
	if mobileStream && !strings.HasPrefix(msg.Message, "!") {
		comm.ToOverlay(fmt.Sprintf("tts false false %s says %s", msg.User.DisplayName, msg.Message))
	}

	if !strings.HasPrefix(msg.Message, "!") {
		return
	}
	msg.Message = handler.InjectAliases(msg.Message)
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))

	// Handle trivial commands here
	switch strings.ToLower(args[0]) {
	case "alias":
		handler.Alias(msg)
		return
	case "clearconsole":
		console.ClearConsole()
		return
	case "commands":
		comm.ToChat(msg.Channel, "See available commands at: https://burtbot.app/commands")
		return
	case "fakefollow":
		if len(args) < 2 {
			return
		}
		FollowAlertToOverlay(args[1])
		return
	case "help":
		handler.HelpAll(msg.Channel)
		return
	case "mobilestream":
		toggleMobileStreamMode(msg)
		return
	case "raidtest":
		if !IsMod(msg.User) {
			return
		}
		comm.ToOverlay("raidincoming person 5")
		return
	case "resetDistance":
		if !IsMod(msg.User) {
			return
		}
		comm.ToOverlay("distance reset")
		return
	case "roll":
		roll(msg)
		return
	case "remind":
		remind(msg)
		return
	}

	// Check the installed commands and execute one if it exists.
	lcmd := strings.ToLower(args[0])
	if cmd, ok := handler.Commands[lcmd]; ok {
		// If a valid command was supplied with help second, show
		// the available help for the command in the chat
		if len(args) > 1 && args[1] == "help" {
			for _, h := range cmd.Help() {
				handler.Client.Say(msg.Channel, h)
			}
		}
		go func() {
			// Recover from any panics that occur in the called command
			// This keeps the entire bot from crashing due to out of range
			// errors and the like inside of commands that may not be fully
			// set up...
			defer func() {
				if r := recover(); r != nil {
					log.Println("Recovered from panic in command handler", r, "from cmd: ", msg.Message)
					comm.ToChat(msg.Channel, "Oh no, burtbot broke burtbot's brian. Maybe use the commands properly not break burtbot's brain.")
				}
			}()
			cmd.Run(msg)
		}()
	}
}

func (handler *CmdHandler) Alias(msg twitch.PrivateMessage) {
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

// Show all the commands help text.. all of them... at once.
// Or say them all????
func (handler *CmdHandler) HelpAll(channel string) {
	if time.Since(lastHelpAll).Seconds() < helpAllCooldown {
		comm.ToChat(channel, "Sorry, I've helped as much as I can for a little while.")
		return
	}
	lastHelpAll = time.Now()
	for _, cmd := range handler.Commands {
		for _, h := range cmd.Help() {
			comm.ToOverlay(fmt.Sprintf("tts true true %s", h))
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
	if _, ok := handler.aliases[alias]; ok {
		return errors.New("alias already exists")
	}
	handler.aliases[alias] = commandName
	handler.saveAliasesToFile()
	return nil
}

func (handler *CmdHandler) RemoveAlias(alias string) bool {
	if _, ok := handler.aliases[alias]; !ok {
		return false
	}
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

func IsMod(user twitch.User) bool {
	_, bcOk := user.Badges["broadcaster"]
	_, modOk := user.Badges["moderator"]
	_, vipOk := user.Badges["vip"]
	return bcOk || modOk || vipOk
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
			Name:    cmdName,
			Help:    []string{},
			Aliases: map[string]string{},
		}
		for k, v := range cmdHandler.aliases {
			if strings.HasPrefix(v, cmdName) {
				c.Aliases[k] = v
			}
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

func FollowAlertToOverlay(username string) {
	if mobileStream {
		comm.ToOverlay(fmt.Sprintf("tts false false %s is now following! Right now, they are following you watch out", username))
	}
	comm.ToOverlay(fmt.Sprintf("newfollow %s", username))
}

func RaidAlertToOverlay(username string, viewers int) {
	comm.ToOverlay(fmt.Sprintf("raidincoming %s %d", username, viewers))
}

// CheckArgs will check to make sure the args slice is at least the correct length
// and that each item is the correct type
func CheckArgs(args []string, count int, argStruct interface{}) (bool, error) {
	// args is the args from the command string
	// first check to see if the length checks out
	if len(args) < count {
		return false, nil
	}

	// We also need to make sure that the argStruct has the same number of elements
	v := reflect.ValueOf(argStruct).Elem()
	if v.NumField() != count {
		return false, errors.New("number of args and fields in struct do not match")
	}

	// Step through the struct
	for i := 0; i < count; i++ {
		// Check the type expected in the struct
		fieldValue := v.Field(i)
		switch fieldValue.Kind() {
		case reflect.Bool:
			b, err := strconv.ParseBool(args[i])
			if err != nil {
				return false, nil
			}
			fieldValue.SetBool(b)
		case reflect.Complex64, reflect.Complex128:
			bitSize := 64
			if fieldValue.Kind() == reflect.Complex128 {
				bitSize = 128
			}
			c, err := strconv.ParseComplex(args[i], bitSize)
			if err != nil {
				return false, nil
			}
			fieldValue.SetComplex(c)
		case reflect.Float32, reflect.Float64:
			bitSize := 32
			if fieldValue.Kind() == reflect.Float64 {
				bitSize = 64
			}
			f, err := strconv.ParseFloat(args[i], bitSize)
			if err != nil {
				return false, nil
			}
			fieldValue.SetFloat(f)
		case reflect.Int:
			n, err := strconv.ParseInt(args[i], 10, 64)
			if err != nil {
				return false, nil
			}
			v.Field(i).SetInt(n)
		case reflect.String:
			fieldValue.SetString(args[i])
		case reflect.Uint:
			n, err := strconv.ParseUint(args[i], 10, 64)
			if err != nil {
				return false, nil
			}
			fieldValue.SetUint(n)
		default:
			return false, nil
		}
		// if the arg at that same index cannot be parsed to that type, return false
		// if it can, store the parsed value in the field
	}
	return true, nil
}

// CheckArgsCB wraps CheckArgs and allows a callback function to passed in to be called
// if the check fails
func CheckArgsCB(args []string, count int, callback func(string), argStruct interface{}) (bool, error) {
	if result, err := CheckArgs(args, count, argStruct); !result {
		callback("not good")
		return false, err
	}
	return true, nil
}

func RegisterCommand(pattern string, c Command) error {
	return cmdHandler.RegisterCommand(pattern, c)
}

func SetMobileStream(b bool) {
	mobileStream = b
}

func remind(msg twitch.PrivateMessage) {
	if !IsMod(msg.User) {
		return
	}
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if len(args) < 3 {
		comm.ToChat(msg.Channel, "Not enough args for the thing you wanted to do which wa")
		return
	}
	duration, err := time.ParseDuration(args[1])
	if err != nil {
		comm.ToChat(msg.Channel, "Duration is an invalid time duration deal with it")
		return
	}
	message := strings.Join(args[2:], " ")
	go func() {
		time.Sleep(duration)
		comm.ToOverlay(fmt.Sprintf("tts false true %s", message))
	}()
}

func roll(msg twitch.PrivateMessage) {
	// just 1-100 for now
	roll := rand.Intn(100) + 1
	comm.ToChat(msg.Channel, fmt.Sprintf("@%s rolled a %d out of 100", msg.User.DisplayName, roll))
}

func toggleMobileStreamMode(msg twitch.PrivateMessage) {
	mobileStream = !mobileStream
	s := "disabled"
	if mobileStream {
		s = "enabled"
	}
	comm.ToChat(msg.Channel, fmt.Sprintf("Mobile stream %s", s))
}
