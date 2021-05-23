package main

import (
	"net/http"
	"fmt"
	"html/template"

	"github.com/MattSwanson/burtbot/commands"
)

var cmdHandler *commands.CmdHandler
var templates *template.Template

func init() {
	templates = template.Must(template.ParseGlob("templates/*"))
}


func StartWebServer(ch *commands.CmdHandler) {

	cmdHandler = ch
	// Add handlers for http stuffs
	http.HandleFunc("/twitch_authcb", commands.TwitchAuthCb)
	http.HandleFunc("/twitch_link", commands.GetAuthLink)
	http.HandleFunc("/eventsub_cb", commands.EventSubCallback)
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
