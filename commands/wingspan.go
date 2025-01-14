package commands

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/MattSwanson/burtbot/db"
	"github.com/MattSwanson/burtbot/web"
	"github.com/gempir/go-twitch-irc/v2"
)

type Wingspan struct {
}

var ws = &Wingspan{}
var unplayedBirdsTpl *template.Template

func init() {
	RegisterCommand("wingspan", ws)
	unplayedBirdsTpl = template.Must(template.ParseFiles("./templates/wingspan.gohtml"))
	web.AuthHandleFunc("/wingspan", unplayedBirds)
	web.AuthHandleFunc("/bird_played", playedBird)
}

func (ws *Wingspan) PostInit() {

}

func (ws *Wingspan) Run(msg twitch.PrivateMessage) {
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if len(args) == 1 {
		return
	}

	if args[1] == "load" && IsMod(msg.User) {
		//db.LoadWingspanDataFromCSV()
		comm.ToChat(msg.Channel, "Loading disabled for WS")
		return
	}

	if args[1] == "unplayed" {
		unplayed, err := db.GetUnplayedBirdCountBySet()
		if err != nil {
			comm.ToChat(msg.Channel, "Error getting count, sorry.")
			return
		}
		comm.ToChat(msg.Channel, "Birds left to play:")
		total := 0
		for set, count := range unplayed {
			total += count
			comm.ToChat(msg.Channel, fmt.Sprintf("%s: %d", set, count))
		}
		comm.ToChat(msg.Channel, fmt.Sprintf("Total: %d", total))
		return
	}
}

func playedBird(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	err = markBirdPlayed(id, true)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "https://burtbot.app/wingspan", http.StatusSeeOther)
}

func markBirdPlayed(birdID int, hasPlayed bool) error {
	err := db.MarkBirdPlayed(birdID, hasPlayed)
	return err
}

func unplayedBirds(w http.ResponseWriter, r *http.Request) {
	birds, err := db.GetUnplayedBirds()
	if err != nil {
		http.Error(w, "Couldn't get birds", http.StatusInternalServerError)
		return
	}
	counts, err := db.GetUnplayedBirdCountBySet()
	if err != nil {
		http.Error(w, "Couldn't get bird counts", http.StatusInternalServerError)
		return
	}
	d := struct {
		Oceania  int
		European int
		Core     int
		Unplayed []db.WingspanBird
	}{
		Oceania:  counts["oceania"],
		European: counts["european"],
		Core:     counts["core"],
		Unplayed: birds,
	}
	unplayedBirdsTpl.ExecuteTemplate(w, "wingspan.gohtml", d)
}

func (ws *Wingspan) Help() []string {
	return []string{
		"WORK IN PROGRESSSES",
	}
}
