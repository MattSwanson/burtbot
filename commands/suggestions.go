package commands

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/MattSwanson/burtbot/db"
	"github.com/MattSwanson/burtbot/web"
	"github.com/gempir/go-twitch-irc/v2"
)

type SuggestionBox struct {
	Suggestions []db.Suggestion
}

var suggestionBox *SuggestionBox = &SuggestionBox{}
var tpl *template.Template

func init() {
	suggestionBox.Suggestions = []db.Suggestion{}
	tpl = template.Must(template.ParseFiles("./templates/suggestions.gohtml"))
	RegisterCommand("sb", suggestionBox)
	web.AuthHandleFunc("/suggestions", showAll)
	web.AuthHandleFunc("/suggestion/delete", deleteSuggestion)
	web.AuthHandleFunc("/suggestion/complete", markComplete)
}

func NewSuggestionBox() *SuggestionBox {
	return suggestionBox
}

func (sb *SuggestionBox) PostInit() {
	suggs, err := db.GetSuggestions()
	if err != nil {
		log.Fatal(err)
	}
	suggestionBox.Suggestions = suggs
}

func (sb *SuggestionBox) Run(msg twitch.PrivateMessage) {
	args := strings.Fields(msg.Message)
	if len(args) < 2 {
		return
	}
	if args[1] == "submit" {
		userID, err := strconv.Atoi(msg.User.ID)
		if err != nil {
			log.Println("got a bad user id: ", err)
			return
		}
		suggestion := db.Suggestion{
			Username: msg.User.DisplayName,
			UserID:   userID,
			Date:     time.Now(),
			Text:     strings.Join(args[2:], " "),
		}
		newId, err := db.AddSuggestion(suggestion)
		if err != nil {
			comm.ToChat(msg.Channel, "Sorry, couldn't add the suggestion at this time. Try again later")
			return
		}
		suggestion.ID = newId
		sb.Suggestions = append(sb.Suggestions, suggestion)
		comm.ToChat(msg.Channel, fmt.Sprintf("Thank you @%s. Your feedback has been noted.", msg.User.DisplayName))
		return
	}
	if args[1] == "complete" && IsMod(msg.User) {
		if len(args) < 3 {
			return
		}
		i, err := strconv.Atoi(args[2])
		if err != nil || i > len(sb.Suggestions) || i < 1 {
			return
		}
		if err = db.SetSuggestionCompletion(sb.Suggestions[i-1].ID, true); err != nil {
			comm.ToChat(msg.Channel, "Couldn't update the DB, sorry. Don't care anymore.")
			return
		}
	}
	if args[1] == "get" {
		if i, err := strconv.Atoi(args[2]); err == nil && i <= len(sb.Suggestions) && i > 0 {
			suggestion := sb.Suggestions[i-1]
			str := fmt.Sprintf("Suggestion #%d from %s:", i, suggestion.Username)
			comm.ToChat(msg.Channel, str)
			complete := ""
			if sb.Suggestions[i-1].Complete {
				complete = "COMPLETE."
			}
			comm.ToChat(msg.Channel, fmt.Sprint(suggestion.Text, complete))
		}
	}
	if args[1] == "count" {
		comm.ToChat(msg.Channel, fmt.Sprintf("There have been %d very good and reasonable suggestions.", len(sb.Suggestions)))
	}

	if args[1] == "all" {
		for _, suggestion := range sb.Suggestions {
			comm.ToOverlay(fmt.Sprintf("tts %s suggested that we %s", suggestion.Username, suggestion.Text))
		}
	}
}

func showAll(w http.ResponseWriter, r *http.Request) {
	suggs, err := db.GetSuggestions()
	if err != nil {
		log.Println(err)
		http.Error(w, "Am teapot", http.StatusTeapot)
		return
	}
	tpl.ExecuteTemplate(w, "suggestions.gohtml", suggs)
}

func deleteSuggestion(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	err = db.DeleteSuggestion(id)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "https://burtbot.app/suggestions", http.StatusSeeOther)
}

func markComplete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	err = db.SetSuggestionCompletion(id, true)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "https://burtbot.app/suggestions", http.StatusSeeOther)
}

func (sb *SuggestionBox) Help() []string {
	return []string{
		"!sb submit [suggestion] to suggest a suggestion",
		"!sb get [number] get a suggestion which has been suggested",
		"!sb count get the number of suggestions which have been suggested thus far",
	}
}
