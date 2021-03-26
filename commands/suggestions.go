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
)

type suggestion struct {
	Username string
	Date     time.Time
	Text     string
}

type SuggestionBox struct {
	Suggestions []suggestion
}

func (sb *SuggestionBox) Init() {
	sb.Suggestions = []suggestion{}
	j, err := os.ReadFile("./suggestions.json")
	if err != nil {
		log.Println("Couldn't loat suggestions from file")
	}
	err = json.Unmarshal(j, &sb.Suggestions)
	if err != nil {
		log.Println("Invalid json in suggestions file")
	}
}

func (sb *SuggestionBox) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	args := strings.Fields(msg.Message)
	if len(args) < 2 {
		return
	}
	if args[1] == "submit" {
		suggestion := suggestion{
			Username: msg.User.DisplayName,
			Date:     time.Now(),
			Text:     strings.Join(args[2:], " "),
		}
		sb.Suggestions = append(sb.Suggestions, suggestion)
		client.Say(msg.Channel, fmt.Sprintf("Thank you @%s. Your feedback has been noted.", msg.User.DisplayName))
		sb.saveToFile()
		return
	}
	if args[1] == "get" {
		if len(args) < 3 {
			return
		}
		if i, err := strconv.Atoi(args[2]); err == nil && len(sb.Suggestions) >= i {
			suggestion := sb.Suggestions[i]
			str := fmt.Sprintf("Suggestion #%d from %s:", i, suggestion.Username)
			client.Say(msg.Channel, str)
			client.Say(msg.Channel, suggestion.Text)
		}
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
}
