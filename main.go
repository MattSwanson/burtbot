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

// var twitchAuthCh chan bool
// var twitchAuth bool
// var twitchAccessToken string
// var twitchRefreshToken string

// type twitchAuthResp struct {
// 	Access_token  string
// 	Refresh_token string
// 	Expires_in    int
// 	Scope         []string
// 	Token_type    string
// }

var bopometer *commands.Bopometer

func main() {

	commChannel = make(chan string)
	readChannel = make(chan string)
	go connectToOverlay()

	client = twitch.NewClient("burtbot11", os.Getenv("BURTBOT_TWITCH_KEY"))
	client.OnPrivateMessage(handleMessage)
	client.OnUserPartMessage(handleUserPart)
	client.OnUserJoinMessage(handleUserJoin)
	client.OnConnect(func() {
		fmt.Println("burtbot circuits activated")
	})

	twitchAuthClient := commands.TwitchAuthClient{}
	go twitchAuthClient.Init()

	client.Join("burtstanton")

	handler = commands.NewCmdHandler(client)
	handler.RegisterCommand("nonillion", commands.Nonillion{})
	handler.RegisterCommand("ded", &commands.Ded{})
	handler.RegisterCommand("oven", &commands.Oven{Temperature: 65, BakeTemp: 0})
	handler.RegisterCommand("bbmsg", &commands.Msg{TcpChannel: commChannel})

	jokes := commands.Joke{TcpChannel: commChannel}
	jokes.Init()
	handler.RegisterCommand("joke", &jokes)
	handler.RegisterCommand("lights", &commands.Lights{})
	handler.RegisterCommand("time", &commands.Tim{})
	handler.RegisterCommand("sb", &commands.SuggestionBox{})

	burtCoin := commands.BurtCoin{}
	burtCoin.Init()
	handler.RegisterCommand("burtcoin", &burtCoin)

	tokenMachine := commands.TokenMachine{BurtCoin: &burtCoin}
	tokenMachine.Init()
	handler.RegisterCommand("tokenmachine", &tokenMachine)

	musicManager := commands.Music{TokenMachine: &tokenMachine}
	go musicManager.Init()
	handler.RegisterCommand("music", &musicManager)

	bbset := commands.Bbset{ReservedCommands: &handler.Commands}
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

	go handleResults(&plinko, &tokenMachine, &snake, &tanks, &bop)

	err := client.Connect()
	if err != nil {
		panic(err)
	}
}

func handleMessage(msg twitch.PrivateMessage) {
	lower := strings.ToLower(msg.Message)
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
	log.Printf(`%s has "parted" the channel.`, msg.User)
	// handle any commands that have interaction with users leaving here
	go handler.HandlePartMsg(msg)
}

func handleUserJoin(msg twitch.UserJoinMessage) {
	log.Printf(`%s has joined the channel.`, msg.User)
}

func connectToOverlay() {
	conn, err := net.Dial("tcp", "localhost:8081")
	if err != nil {
		log.Println("Couldn't connect to overlay")
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
		fmt.Println(s)
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
				plural := ""
				if n > 1 {
					plural = "s"
				}
				s := fmt.Sprintf("@%s won %d token%s!", args[2], n, plural)
				client.Say("burtstanton", s)
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
