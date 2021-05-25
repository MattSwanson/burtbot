package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
	"github.com/MattSwanson/burtbot/db"
	"github.com/MattSwanson/burtbot/commands"
	"github.com/gempir/go-twitch-irc/v2"
)

var handler *commands.CmdHandler
var client *twitch.Client
var bbset commands.Bbset

var lastMessage string
var lastMsg twitch.PrivateMessage

var schlorpLock = false
var schlorpCD = 10

var commChannel chan string
var readChannel chan string

var bopometer *commands.Bopometer

func main() {

	commChannel = make(chan string)
	readChannel = make(chan string)
	go connectToOverlay()

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
	client.OnConnect(func() {
		fmt.Println("burtbot circuits activated")
	})

	handler = commands.NewCmdHandler(client, commChannel)

	burtCoin := commands.BurtCoin{}
	burtCoin.Init()
	handler.RegisterCommand("burtcoin", &burtCoin)

	tokenMachine := commands.TokenMachine{BurtCoin: &burtCoin}
	tokenMachine.Init()
	handler.RegisterCommand("tokenmachine", &tokenMachine)

	twitchAuthClient := commands.TwitchAuthClient{}
	go twitchAuthClient.Init(client, &tokenMachine)

	client.Join("burtstanton")

	//handler.RegisterCommand("nonillion", commands.Nonillion{})
	handler.RegisterCommand("ded", &commands.Ded{})
	handler.RegisterCommand("oven", &commands.Oven{Temperature: 65, BakeTemp: 0})
	handler.RegisterCommand("bbmsg", &commands.Msg{TcpChannel: commChannel})
	handler.RegisterCommand("offbyone", &commands.OffByOneCounter{})

	jokes := commands.Joke{TcpChannel: commChannel}
	jokes.Init()
	handler.RegisterCommand("joke", &jokes)
	handler.RegisterCommand("lights", &commands.Lights{})
	handler.RegisterCommand("time", &commands.Tim{})
	sb := commands.NewSuggestionBox()
	//sb.Init()
	handler.RegisterCommand("sb", sb)

	musicManager := commands.Music{TokenMachine: &tokenMachine}
	go musicManager.Init()
	handler.RegisterCommand("music", &musicManager)

	bbset = commands.Bbset{ReservedCommands: &handler.Commands}
	bbset.Init()
	handler.RegisterCommand("bbset", &bbset)

	bop := commands.Bopometer{Music: &musicManager, TCPChannel: commChannel}
	bop.Init()
	handler.RegisterCommand("bop", &bop)
	bopometer = &bop

	goph := commands.Gopher{TcpChannel: commChannel}
	handler.RegisterCommand("go", &goph)

	bigMouse := commands.BigMouse{TcpChannel: commChannel}
	handler.RegisterCommand("bigmouse", &bigMouse)

	snake := commands.Snake{TcpChannel: commChannel}
	handler.RegisterCommand("snake", &snake)

	marquee := commands.Marquee{TcpChannel: commChannel}
	handler.RegisterCommand("marquee", &marquee)

	handler.RegisterCommand("so", &commands.Shoutout{TcpChannel: commChannel, TwitchClient: &twitchAuthClient})

	plinko := commands.Plinko{TcpChannel: commChannel, TokenMachine: &tokenMachine}
	handler.RegisterCommand("plinko", &plinko)

	tanks := commands.Tanks{TcpChannel: commChannel}
	handler.RegisterCommand("tanks", &tanks)

	lightsOut := commands.LightsOut{CommChannel: commChannel}
	handler.RegisterCommand("lo", &lightsOut)

	handler.RegisterCommand("protocolr", &commands.ProtoR{})
	handler.RegisterCommand("incomplete", &commands.Incomplete{})
	bingo := commands.NewBingo(&twitchAuthClient, &tokenMachine, commChannel)
	handler.RegisterCommand("bingo", bingo)
	//importSuggestions(&twitchAuthClient, sb.Suggestions)

	go handleResults(&plinko, &tokenMachine, &snake, &tanks, &bop)
	StartWebServer(handler)
	
	err = client.Connect()
	if err != nil {
		panic(err)
	}
}

func handleMessage(msg twitch.PrivateMessage) {
	if msg.User.DisplayName == "tundragaminglive" {
		commChannel <- "miracle"
	}
	lower := strings.ToLower(msg.Message)
	if lower == "!help" {
		handler.HelpAll()
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
		commChannel <- "up"
	}
	if lower == "a" {
		commChannel <- "left"
	}
	if lower == "s" {
		commChannel <- "down"
	}
	if lower == "d" {
		commChannel <- "right"
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
	// if strings.Contains(lower, "quack") {
	// 	commChannel <- "quack"
	// }
	if count := strings.Count(lower, "quack"); count > 0 {
		commChannel <- fmt.Sprintf("quack %d", count)
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

func connectToOverlay() {
	addr := fmt.Sprintf("%s:8081", os.Getenv("OVERLAY_IP"))
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		//log.Println("Couldn't connect to overlay")
		time.Sleep(time.Second * 10)
		connectToOverlay()
		return
	}
	defer conn.Close()
	go func() {
		getMessagesFromTCP(conn)
	}()
	ctx, cancelPing := context.WithCancel(context.Background())
	pingOverlay(ctx, commChannel)
	fmt.Println("Connected to overlay")
	for {
		s := <-commChannel
		// fmt.Println(s)
		_, err := fmt.Fprintf(conn, "%s\n", s)
		if err != nil {
			// we know we have no connection, stop pinging until we reconnect
			cancelPing()
			log.Println("Lost connection to overlay... will retry in 5 sec.")
			readChannel <- "reset"
			time.Sleep(time.Second * 5)
			connectToOverlay()
		}
	}
}

func getMessagesFromTCP(conn net.Conn) {
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		s := scanner.Text()
		fmt.Println(s)
		readChannel <- s
	}
	if err := scanner.Err(); err != nil {
		log.Println(err)
	}
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
			if n, err := strconv.Atoi(args[3]); err == nil {
				t.GrantToken(strings.ToLower(args[2]), n)

				s := ""
				if n > 0 {
					plural := ""
					if n > 1 {
						plural = "s"
					}
					s = fmt.Sprintf("@%s won %d token%s!", args[2], n, plural)
				} else {
					s = fmt.Sprintf("@%s, YOU GET NOTHING! GOOD DAY!", args[2])
				}
				//client.Say("burtstanton", s)
				mMsg := commands.MarqueeMsg{
					RawMessage: s,
					Emotes:     "",
				}
				json, err := json.Marshal(mMsg)
				if err != nil {
					return
				}
				commChannel <- "marquee once " + string(json)
			}
			//p.Stop
		case "bop":
			b.Results(client, args[2])
		case "reset":
			snake.SetRunning(false)
			p.Stop()
			tanks.Stop()
		}
	}
}

// put a ping on the comm channel every 3 seconds to make sure we still have a connection
func pingOverlay(ctx context.Context, c chan string) {
	go func(ctx context.Context) {
		t := time.NewTicker(time.Second * 3)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				c <- "ping"
			}
		}
	}(ctx)
}

func unlockSchlorp() {
	time.Sleep(time.Second * time.Duration(schlorpCD))
	schlorpLock = false
	log.Println("schlorp unlocked")
}
