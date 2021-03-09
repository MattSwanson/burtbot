package commands

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gempir/go-twitch-irc/v2"
)

type Joke struct{}

type apiResponse struct {
	ID     string `json:"id"`
	Joke   string `json:"joke"`
	Status int    `json:"status"`
}

var jokeLock bool = false
var jokeCD int = 180 // seconds

func (j Joke) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	if jokeLock && !isMod(msg.User) {
		return
	}
	if !jokeLock {
		jokeLock = true
		go unlockJoke()
	}
	// Fetch a joke from icanhazdadjoke api
	req, err := http.NewRequest("GET", "https://icanhazdadjoke.com", nil)
	if err != nil {
		client.Say(msg.Channel, "Sorry, couldn't connect to joke dispensory")
		log.Println("Couldn't access joke api: ", err.Error())
		return
	}
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	//resp, err := http.Get("https://icanhazdadjoke.com/")
	if err != nil {
		client.Say(msg.Channel, "Sorry, couldn't connect to joke dispensory")
		log.Println("Couldn't access joke api: ", err.Error())
		return
	}
	r := apiResponse{}
	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		client.Say(msg.Channel, "I dropped the joke on the way back from the store, sorry.")
		log.Println(err.Error())
		return
	}

	// send the joke to the message speaker
	_, err = http.PostForm("http://localhost:8080/sendMessage", url.Values{"message": {r.Joke}})
	if err != nil {
		log.Println(err.Error())
		client.Say(msg.Channel, "I'm having trouble speaking but...")
	}
	// Some jokes have \r\n in them - I think we need to filter those out
	jokes := strings.Split(r.Joke, "\n")
	for _, joke := range jokes {
		client.Say(msg.Channel, joke)
	}

}

func unlockJoke() {
	time.Sleep(time.Second * time.Duration(jokeCD))
	jokeLock = false
}

func manualUnlock() {

}
