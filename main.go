package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	"github.com/MattSwanson/burtbot/db"
	"github.com/MattSwanson/burtbot/commands"
	"github.com/MattSwanson/burtbot/comm"
	"github.com/MattSwanson/burtbot/helix"
	"github.com/gempir/go-twitch-irc/v2"
)

var handler *commands.CmdHandler
var client *twitch.Client
var chatMessages []twitch.PrivateMessage

var lastMessage string
var lastMsg twitch.PrivateMessage 
var lastQuacksplosion time.Time
var schlorpLock = false
var schlorpCD = 10


func main() {

	go comm.ConnectToOverlay()

	// init db connection
	err, closeDb := db.Connect()
	if err != nil {
		log.Fatalln("failed to connect to db: ", err)
	}
	defer closeDb()

	client = twitch.NewClient("burtbot11", os.Getenv("BURTBOT_TWITCH_KEY"))
	client.OnPrivateMessage(handleMessage)
	client.OnUserPartMessage(handleUserPart)
	client.OnUserJoinMessage(handleUserJoin)
	client.OnClearMessage(handleClearMessage)
	client.OnClearChatMessage(handleClearChatMessage)
	client.OnConnect(func() {
		fmt.Println("burtbot circuits activated")
	})

	helix.Init()

	handler = commands.NewCmdHandler(client)
	handler.PostInit()
	handler.LoadAliases()
	client.Join("burtstanton")
	comm.AddChatClient(client)
	StartWebServer(handler)
	
	err = client.Connect()
	if err != nil {
		panic(err)
	}
}

func handleMessage(msg twitch.PrivateMessage) {

	showMessageOnConsole(msg)
	
	if msg.User.DisplayName == "tundragaminglive" {
		comm.ToOverlay("miracle")
	}

	lower := strings.ToLower(msg.Message)
	if lower == "!help" {
		handler.HelpAll()
	}
	msg.Message = handler.InjectAliases(msg.Message)

	fields := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if commands.IsMod(msg.User) && fields[0] == "alias" && len(fields) >= 4 {
		// !alias add alias command
		if fields[1] == "add" {
			originalCommand := strings.Join(fields[3:], " ")
			err := handler.RegisterAlias(fields[2], originalCommand)			
			if err != nil {
				comm.ToChat(msg.Channel, fmt.Sprintf("The alias [%s] already exists.", fields[2]))
				return
			}
			comm.ToChat(msg.Channel, fmt.Sprintf("Created alias [%s] for [%s]", fields[2], originalCommand)) 
			return
		}
	}

	if lower == "!commands" {
		comm.ToChat(msg.Channel, "See available commands at: https://burtbot.app/commands")
		return
	}

	if lower == "w" {
		comm.ToOverlay("up")
	}
	if lower == "a" {
		comm.ToOverlay("left")
	}
	if lower == "s" {
		comm.ToOverlay("down")
	}
	if lower == "d" {
		comm.ToOverlay("right")
	}


	go handler.HandleMsg(msg)
	if strings.Compare(msg.User.Name, lastMsg.User.Name) == 0 && strings.Compare(msg.Message, lastMessage+" "+lastMessage) == 0 {
		// break the pyramid with a schlorp
		comm.ToChat(msg.Channel, "tjportSchlorp1 tjportSchlorp2 tjportSchlorp3")
	}
	lower = strings.ToLower(msg.Message)
	if strings.Contains(lower, "schlorp") {
		if !schlorpLock {
			schlorpLock = true
			go unlockSchlorp()
			comm.ToChat(msg.Channel, "tjportSchlorp1 tjportSchlorp2 tjportSchlorp3")
		}
	}
	if strings.Contains(lower, "one time") {
		comm.ToChat(msg.Channel, "ONE TIME!")
	}
	if count := strings.Count(lower, "quack"); count > 0 {
		comm.ToOverlay(fmt.Sprintf("quack %d", count))
		if msg.User.DisplayName == "0xffffffff810000000" {
			if time.Since(lastQuacksplosion).Seconds() > 21600 {
				comm.ToOverlay("quacksplosion")
				lastQuacksplosion = time.Now()
			}
		}
	}

	lastMessage = msg.Message
	lastMsg = msg
}

//TODO: Figure out if this can work consistantly enough for what we want it for
func handleUserPart(msg twitch.UserPartMessage) {
	//log.Printf(`%s has "parted" the channel.`, msg.User)
	// handle any commands that have interaction with users leaving here
	go handler.HandlePartMsg(msg)
}

func handleUserJoin(msg twitch.UserJoinMessage) {
	//log.Printf(`%s has joined the channel.`, msg.User)
	go handler.HandleJoinMsg(msg)
}

func unlockSchlorp() {
	time.Sleep(time.Second * time.Duration(schlorpCD))
	schlorpLock = false
}
