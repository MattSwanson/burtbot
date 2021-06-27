package main

import (
	"net/http"
	"strings"
	"os"
	"fmt"
	"html/template"

	"github.com/MattSwanson/burtbot/commands"
	"github.com/MattSwanson/burtbot/helix"
)

var cmdHandler *commands.CmdHandler
var templates *template.Template
var serviceAuthStatus *ServiceAuthStatus

type ServiceAuthStatus struct {
	SpotifyAuth bool
	SpotifyLink string
	TwitchAuth bool
	TwitchLink string
}


func init() {
	templates = template.Must(template.ParseGlob("templates/*"))
	serviceAuthStatus = &ServiceAuthStatus{}
}


func StartWebServer(ch *commands.CmdHandler) {

	cmdHandler = ch
	// Add handlers for http stuffs
	http.HandleFunc("/services_auth", servicesAuthPage)
	http.HandleFunc("/twitch_authcb", helix.TwitchAuthCb)
	http.HandleFunc("/eventsub_cb", helix.EventSubCallback)
	http.HandleFunc("/commands", commandList)
	http.HandleFunc("/bingo", commands.DisplayCards)
	http.HandleFunc("/", home)

	// Create a web server to listen on HTTPS
	go http.ListenAndServeTLS(":443", "/etc/letsencrypt/live/burtbot.app/fullchain.pem", "/etc/letsencrypt/live/burtbot.app/privkey.pem", nil)

}

func home(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	fmt.Fprint(w, "boop.\n")
}

type cmdHelp struct {
	Name string
	Help []string
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
	err := templates.ExecuteTemplate(w, "help.gohtml", cmds)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func servicesAuthPage(w http.ResponseWriter, r *http.Request) {
	remote := strings.Split(r.RemoteAddr, ":")
	if remote[0] != os.Getenv("OVERLAY_IP") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	serviceAuthStatus.TwitchAuth = helix.GetAuthStatus()
	if !serviceAuthStatus.TwitchAuth {
		serviceAuthStatus.TwitchLink = helix.GetAuthLink()
	}
	serviceAuthStatus.SpotifyAuth = commands.GetSpotifyAuthStatus()
	if !serviceAuthStatus.SpotifyAuth {
		serviceAuthStatus.SpotifyLink = commands.GetSpotifyLink()
	}
	templates.ExecuteTemplate(w, "serviceAuthPage.gohtml", serviceAuthStatus)
}
