package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/MattSwanson/burtbot/commands"
	"github.com/gempir/go-twitch-irc/v2"
	"github.com/zmb3/spotify"
)

var handler *commands.CmdHandler
var client *twitch.Client
var bbset commands.Bbset

var lastMessage string
var lastMsg twitch.PrivateMessage

var schlorpLock = false
var schlorpCD = 10

var spotifyAuth = spotify.NewAuthenticator("http://localhost:8079/spotify_authcb",
	spotify.ScopeUserReadPrivate,
	spotify.ScopeUserReadCurrentlyPlaying,
	spotify.ScopeUserReadRecentlyPlayed,
	spotify.ScopeUserModifyPlaybackState)
var spotifyAuthCh = make(chan *spotify.Client)
var spotifyState = "test123"

func main() {

	//TODO
	// We need to update the commands framework to allow initializations on a per module basis
	// like all of the spotify init should be in the music command
	// and all of bbset's junk etc.

	// init and authorize spotify stuff
	http.HandleFunc("/spotify_authcb", completeAuth)
	go http.ListenAndServe(":8079", nil)

	url := spotifyAuth.AuthURL(spotifyState)
	fmt.Println("Auth url for spotify: ", url)

	spotifyClient := <-spotifyAuthCh

	user, err := spotifyClient.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Logged in to Spotify as: ", user.ID)
	client = twitch.NewClient("burtbot11", os.Getenv("BURTBOT_TWITCH_KEY"))
	client.OnPrivateMessage(handleMessage)
	client.OnConnect(func() {
		fmt.Println("burtbot circuits activated")
	})

	client.Join("burtstanton")

	handler = commands.NewCmdHandler(client)
	handler.RegisterCommand("nonillion", commands.Nonillion{})
	handler.RegisterCommand("ded", commands.Ded{})
	handler.RegisterCommand("oven", &commands.Oven{Temperature: 65, BakeTemp: 0})
	handler.RegisterCommand("bbmsg", commands.Msg{})
	handler.RegisterCommand("joke", commands.Joke{})
	handler.RegisterCommand("lights", commands.Lights{})
	handler.RegisterCommand("time", commands.Tim{})

	// burtcoin init
	wallets := make(map[string]int)
	j, err := os.ReadFile("./wallets.json")
	if err != nil {
		log.Println("Couldn't load burtcoin wallet info from file")
	} else {
		err = json.Unmarshal(j, &wallets)
		if err != nil {
			log.Println("Invalid json in tokens file")
		}
	}
	burtCoin := &commands.BurtCoin{Wallets: wallets}
	handler.RegisterCommand("burtcoin", burtCoin)

	// tokens init
	tokens := make(map[string]int)
	persistTokens := true
	j, err = os.ReadFile("./tokens.json")
	if err != nil {
		log.Println("Couldn't load token info from file")
		persistTokens = false
	} else {
		err = json.Unmarshal(j, &tokens)
		if err != nil {
			log.Println("Invalid json in tokens file")
			persistTokens = false
		}
	}

	musicManager := &commands.Music{spotifyClient, tokens, persistTokens}
	handler.RegisterCommand("music", musicManager)
	handler.RegisterCommand("tokenmachine", &commands.TokenMachine{Music: musicManager, BurtCoin: burtCoin})

	// A lot of init for bbset should have it's own init func...
	chatCommands := make(map[string]string)
	persistBbset := true
	j, err = os.ReadFile("./commands.json")
	if err != nil {
		log.Println("Couldn't loat chat commands from file")
		persistBbset = false
	}
	err = json.Unmarshal(j, &chatCommands)
	if err != nil {
		log.Println("Invalid json in chat commands file")
		persistBbset = false
	}
	bbset = commands.Bbset{chatCommands, &handler.Commands, persistBbset}
	handler.RegisterCommand("bbset", bbset)

	err = client.Connect()
	if err != nil {
		panic(err)
	}
}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := spotifyAuth.Token(spotifyState, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != spotifyState {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, spotifyState)
	}
	// use the token to get an authenticated client
	client := spotifyAuth.NewClient(tok)
	fmt.Fprintf(w, "Login completed!")
	spotifyAuthCh <- &client
}

func handleMessage(msg twitch.PrivateMessage) {
	go handler.HandleMsg(msg)
	go bbset.HandleMsg(client, msg)
	if strings.Compare(msg.User.Name, lastMsg.User.Name) == 0 && strings.Compare(msg.Message, lastMessage+" "+lastMessage) == 0 {
		// break the pyramid with a schlorp
		client.Say(msg.Channel, "tjportSchlorp1 tjportSchlorp2 tjportSchlorp3")
	}
	lower := strings.ToLower(msg.Message)
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

	lastMessage = msg.Message
	lastMsg = msg
}

func unlockSchlorp() {
	time.Sleep(time.Second * time.Duration(schlorpCD))
	schlorpLock = false
	log.Println("schlorp unlocked")
}
