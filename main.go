package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/MattSwanson/burtbot/commands"
	"github.com/MattSwanson/burtbot/console"
	"github.com/MattSwanson/burtbot/db"
	"github.com/MattSwanson/burtbot/helix"
	"github.com/MattSwanson/burtbot/web"
	"github.com/gempir/go-twitch-irc/v2"
)

var handler *commands.CmdHandler
var client *twitch.Client
var serviceAuthStatus *ServiceAuthStatus
var servicePageTpl *template.Template
var logFile *os.File
var mobileStream bool

func init() {
	servicePageTpl = template.Must(template.ParseFiles("templates/serviceAuthPage.gohtml"))
	var err error
	logFile, err = os.OpenFile("bb_log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("couldn't open log file for writting")
	}
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetOutput(logFile)
}

func main() {
	defer logFile.Close()
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
	client.OnClearMessage(console.HandleClearMessage)
	client.OnClearChatMessage(console.HandleClearChatMessage)
	client.OnConnect(func() {
		console.AddMessage("burtbot circuits activated", console.Red)
	})

	helix.Init()

	handler = commands.NewCmdHandler(client)
	handler.PostInit()
	handler.LoadAliases()
	client.Join("burtstanton")
	comm.AddChatClient(client)
	serviceAuthStatus = &ServiceAuthStatus{}
	web.AuthHandleFunc("/services_auth", servicesAuthPage)
	web.AuthHandleFunc("/toggle_mobile", toggleMobileStream)
	web.StartWebServer()

	err = client.Connect()
	if err != nil {
		panic(err)
	}
}

func handleMessage(msg twitch.PrivateMessage) {
	console.ShowMessageOnConsole(msg)
	if mobileStream {
		toSay := fmt.Sprintf("%s says %s", msg.User.DisplayName, msg.Message)
		comm.ToOverlay(fmt.Sprintf("tts false %s", toSay))
	}
	go handler.HandleMsg(msg)
}

func handleUserPart(msg twitch.UserPartMessage) {
	go handler.HandlePartMsg(msg)
}

func handleUserJoin(msg twitch.UserJoinMessage) {
	go handler.HandleJoinMsg(msg)
}

type ServiceAuthStatus struct {
	SpotifyAuth  bool
	SpotifyLink  string
	TwitchAuth   bool
	TwitchLink   string
	MobileStream bool
}

func servicesAuthPage(w http.ResponseWriter, r *http.Request) {
	serviceAuthStatus.TwitchAuth = helix.GetAuthStatus()
	if !serviceAuthStatus.TwitchAuth {
		serviceAuthStatus.TwitchLink = helix.GetAuthLink()
	}
	serviceAuthStatus.SpotifyAuth = commands.GetSpotifyAuthStatus()
	if !serviceAuthStatus.SpotifyAuth {
		serviceAuthStatus.SpotifyLink = commands.GetSpotifyLink()
	}
	serviceAuthStatus.MobileStream = mobileStream
	servicePageTpl.ExecuteTemplate(w, "serviceAuthPage.gohtml", serviceAuthStatus)
}

func toggleMobileStream(w http.ResponseWriter, r *http.Request) {
	mobileStream = !mobileStream
	servicesAuthPage(w, r)
}
