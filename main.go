package main

import (
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
	web.AuthHandleFunc("/web_command", processWebCommand)
	web.StartWebServer()

	err = client.Connect()
	if err != nil {
		panic(err)
	}
}

func handleMessage(msg twitch.PrivateMessage) {
	console.ShowMessageOnConsole(msg)
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
	commands.SetMobileStream(mobileStream)
	servicesAuthPage(w, r)
}

func processWebCommand(w http.ResponseWriter, r *http.Request) {
	// Get a command from the request form
	// Decide what to do with it
	// Reload the commands page until we do something better..
	if err := r.ParseForm(); err != nil {
		log.Println("Error parsing form in web command: ", err.Error())
		w.WriteHeader(500)
		return
	}
	// Perhaps we should just construct a fake twitch message and send it to the command handler instead of
	// sending commands to the overlay directly
	comm.ToOverlay("steam random")
	servicesAuthPage(w, r)
}
