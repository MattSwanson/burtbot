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
	"github.com/gempir/go-twitch-irc/v2"
)

var handler *commands.CmdHandler
var client *twitch.Client
var bbset commands.Bbset
var chatMessages []twitch.PrivateMessage
var triviaManager *commands.Trivia

var lastMessage string
var lastMsg twitch.PrivateMessage

var schlorpLock = false
var schlorpCD = 10

var readChannel chan string

var bopometer *commands.Bopometer

func main() {

	readChannel = comm.GetReadChannel()
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
	
	registerCommands()
	client.Join("burtstanton")
	StartWebServer(handler)
	
	err = client.Connect()
	if err != nil {
		panic(err)
	}
}

func registerCommands() {
	handler = commands.NewCmdHandler(client)

	burtCoin := commands.BurtCoin{}
	burtCoin.Init()
	handler.RegisterCommand("burtcoin", &burtCoin)

	tokenMachine := commands.TokenMachine{BurtCoin: &burtCoin}
	tokenMachine.Init()
	handler.RegisterCommand("tokenmachine", &tokenMachine)

	twitchAuthClient := commands.TwitchAuthClient{}
	go twitchAuthClient.Init(client, &tokenMachine)

	//handler.RegisterCommand("nonillion", commands.Nonillion{})
	handler.RegisterCommand("ded", &commands.Ded{})
	handler.RegisterCommand("oven", &commands.Oven{Temperature: 65, BakeTemp: 0})
	handler.RegisterCommand("bbmsg", &commands.Msg{})
	handler.RegisterCommand("offbyone", &commands.OffByOneCounter{})

	jokes := commands.Joke{}
	jokes.Init()
	handler.RegisterCommand("joke", &jokes)
	
	handler.RegisterCommand("lights", commands.NewLights())
	handler.RegisterCommand("time", &commands.Tim{})
	handler.RegisterCommand("sb", commands.NewSuggestionBox())

	musicManager := commands.Music{TokenMachine: &tokenMachine}
	go musicManager.Init()
	handler.RegisterCommand("music", &musicManager)

	bbset = commands.Bbset{ReservedCommands: &handler.Commands}
	bbset.Init()
	handler.RegisterCommand("bbset", &bbset)

	bop := commands.Bopometer{Music: &musicManager}
	bop.Init()
	handler.RegisterCommand("bop", &bop)
	bopometer = &bop

	handler.RegisterCommand("go", &commands.Gopher{})
	handler.RegisterCommand("bigmouse", &commands.BigMouse{})

	snake := commands.Snake{}
	handler.RegisterCommand("snake", &snake)
	handler.RegisterCommand("marquee", &commands.Marquee{})
	handler.RegisterCommand("so", &commands.Shoutout{TwitchClient: &twitchAuthClient})
	handler.RegisterCommand("error", &commands.ErrorBox{})

	plinko := commands.Plinko{TokenMachine: &tokenMachine}
	handler.RegisterCommand("plinko", &plinko)

	tanks := commands.Tanks{}
	handler.RegisterCommand("tanks", &tanks)
	handler.RegisterCommand("lo", &commands.LightsOut{})

	triviaManager = commands.NewTrivia()
	handler.RegisterCommand("trivia", triviaManager)

	handler.RegisterCommand("wod", &commands.Wod{})
	handler.RegisterCommand("protocolr", &commands.ProtoR{})
	handler.RegisterCommand("incomplete", &commands.Incomplete{})
	bingo := commands.NewBingo(&twitchAuthClient, &tokenMachine)
	handler.RegisterCommand("bingo", bingo)
	//importSuggestions(&twitchAuthClient, sb.Suggestions)

	handler.LoadAliases()
	go handleResults(&plinko, &tokenMachine, &snake, &tanks, &bop)

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
	go func(){
		if triviaManager.AnswerChannel != nil {
			triviaManager.AnswerChannel <- msg
		}
	}()
	msg.Message = handler.InjectAliases(msg.Message)

	fields := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if commands.IsMod(msg.User) && fields[0] == "alias" && len(fields) >= 4 {
		// !alias add alias command
		if fields[1] == "add" {
			originalCommand := strings.Join(fields[3:], " ")
			err := handler.RegisterAlias(fields[2], originalCommand)			
			if err != nil {
				client.Say(msg.Channel, fmt.Sprintf("The alias [%s] already exists.", fields[2]))
				return
			}
			client.Say(msg.Channel, fmt.Sprintf("Created alias [%s] for [%s]", fields[2], originalCommand)) 
			return
		}
	}

	if lower == "!commands" {
		client.Say(msg.Channel, "See available commands at: https://burtbot.app/commands")
		return
	}

	if bopometer.GetBopping() {
		bops := strings.Count(msg.Message, "BOP")
		bopometer.AddBops(bops)
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
	go bbset.HandleMsg(client, msg)
	if strings.Compare(msg.User.Name, lastMsg.User.Name) == 0 && strings.Compare(msg.Message, lastMessage+" "+lastMessage) == 0 {
		// break the pyramid with a schlorp
		client.Say(msg.Channel, "tjportSchlorp1 tjportSchlorp2 tjportSchlorp3")
	}
	lower = strings.ToLower(msg.Message)
	if strings.Contains(lower, "schlorp") {
		if !schlorpLock {
			schlorpLock = true
			go unlockSchlorp()
			client.Say(msg.Channel, "tjportSchlorp1 tjportSchlorp2 tjportSchlorp3")
		}
	}
	if strings.Contains(lower, "one time") {
		client.Say(msg.Channel, "ONE TIME!")
	}
	if count := strings.Count(lower, "quack"); count > 0 {
		comm.ToOverlay(fmt.Sprintf("quack %d", count))
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
}



func handleResults(
	p *commands.Plinko,
	t *commands.TokenMachine,
	snake *commands.Snake,
	tanks *commands.Tanks,
	b *commands.Bopometer) {
	for s := range readChannel {
		args := strings.Fields(s)
		switch args[0] {
		case "plinko":
			// plinko result username n
			//p.Stop
			p.HandleResponse(args)
		case "bop":
			b.Results(client, args[2])
		case "reset":
			snake.SetRunning(false)
			p.Stop()
			tanks.Stop()
		}
	}
}


func unlockSchlorp() {
	time.Sleep(time.Second * time.Duration(schlorpCD))
	schlorpLock = false
}
