package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gempir/go-twitch-irc/v2"
	"github.com/MattSwanson/burtbot/db"
)


type SuggestionBox struct {
	Suggestions []db.Suggestion
}

func NewSuggestionBox() *SuggestionBox {
	suggestions := []db.Suggestion{}
	j, err := os.ReadFile("./suggestions.json")
	if err != nil {
		log.Println("Couldn't loat suggestions from file")
	}
	err = json.Unmarshal(j, &suggestions)
	if err != nil {
		log.Println("Invalid json in suggestions file")
	}
	return &SuggestionBox{
		Suggestions: suggestions,
	}
}

func (sb *SuggestionBox) Init() {
}

func (sb *SuggestionBox) Run(client *twitch.Client, msg twitch.PrivateMessage) {
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
		sb.Suggestions = append(sb.Suggestions, suggestion)
		db.AddSuggestion(suggestion)
		client.Say(msg.Channel, fmt.Sprintf("Thank you @%s. Your feedback has been noted.", msg.User.DisplayName))
		sb.saveToFile()
		return
	}
	if args[1] == "get" {
		if len(args) < 3 {
			suggs, err := db.GetSuggestions()
			if err != nil {
				client.Say(msg.Channel, "Sorry, can't help you there. Something odd happened")
				log.Println("couldn't get all suggestions: ", err)
				return
			}
			for k, s := range suggs {
				client.Say(msg.Channel, fmt.Sprintf("%d. %s", k+1, s.Text))
			}
			return
		}
		if i, err := strconv.Atoi(args[2]); err == nil && i <= len(sb.Suggestions) && i > 0 {
			suggestion := sb.Suggestions[i-1]
			str := fmt.Sprintf("Suggestion #%d from %s:", i, suggestion.Username)
			client.Say(msg.Channel, str)
			client.Say(msg.Channel, suggestion.Text)
		}
	}
	if args[1] == "count" {
		client.Say(msg.Channel, fmt.Sprintf("There have been %d very good and reasonable suggestions.", len(sb.Suggestions)))
	}
}

func (sb *SuggestionBox) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {

}

func (sb *SuggestionBox) saveToFile() {
	json, err := json.Marshal(sb.Suggestions)
	if err != nil {
		log.Println("Couldn't json")
		return
	}
	if err := os.WriteFile("./suggestions.json", json, 0644); err != nil {
		log.Println(err.Error())
	}

	/*for _, s := range sb.Suggestions {
		err := db.AddSuggestion(s)
		if err != nil {
			log.Println("unable to write suggestion to db", err)
			return
		}
	}*/
}

func (sb *SuggestionBox) Help() []string {
	return []string{
		"!sb submit [suggestion] to suggest a suggestion",
		"!sb get [number] get a suggestion which has been suggested",
		"!sb count get the number of suggestions which have been suggested thus far",
	}
}
