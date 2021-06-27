package web

import (
	"net/http"
	"strings"
	"os"
	"fmt"
	"html/template"

	"github.com/MattSwanson/burtbot/helix"
)

var templates *template.Template


func init() {
	templates = template.Must(template.ParseGlob("templates/*"))
}


func StartWebServer() {

	// Add handlers for http stuffs
	http.HandleFunc("/twitch_authcb", helix.TwitchAuthCb)
	http.HandleFunc("/eventsub_cb", helix.EventSubCallback)
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



func AuthHandleFunc(pattern string, handlerFunc func(http.ResponseWriter, *http.Request)) {
	wrappedFunc := func(w http.ResponseWriter, r *http.Request) {
		// run our auth on the request
		if !authenticateRequest(r) {
		// if no go - retrun StatusForbidden
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		// otherwise run our handlerFunc
		handlerFunc(w, r)
	}
	http.HandleFunc(pattern, wrappedFunc)
}

// authenticateRequest will check the request to make sure it's legit. Returns true if
// everything checks out - false for any other situation including errors
//TODO needs to be turned into actual auth - just checking against remote address at
// this point
func authenticateRequest(r *http.Request) bool {
	remote := strings.Split(r.RemoteAddr, ":")
	if remote[0] != os.Getenv("OVERLAY_IP") {
		return false
	}
	return true
}

