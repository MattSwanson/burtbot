package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gempir/go-twitch-irc/v2"
)

type Joke struct {
	TcpChannel   chan string
	jokeMode     bool
	jokeModeStop chan bool
}

type apiResponse struct {
	ID     string `json:"id"`
	Joke   string `json:"joke"`
	Status int    `json:"status"`
}

var jokeLock bool = false
var jokeCD int = 30 // seconds

func (j *Joke) Init() {
	j.jokeModeStop = make(chan bool)
	j.jokeMode = false
}

func (j *Joke) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	if jokeLock && !isMod(msg.User) {
		return
	}
	if !jokeLock {
		jokeLock = true
		go unlockJoke()
	}
	args := strings.Fields(strings.ToLower(strings.TrimPrefix(msg.Message, "!")))
	if len(args) == 1 {
		j.TellJoke(client, msg)
		return
	}

	if args[1] == "mode" && isMod(msg.User) {
		if len(args) < 3 {
			return
		}
		// start
		if args[2] == "start" && !j.jokeMode {
			client.Say(msg.Channel, "Initiating joke mode - prepare for copious amounts of laughter.")
			j.JokeMode(client, msg)
		}
		// stop
		if args[2] == "stop" && j.jokeMode {
			client.Say(msg.Channel, "Ending joke mode - try to stop laughing now.")
			j.jokeModeStop <- true
		}
	}
}

func (j *Joke) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {

}

func unlockJoke() {
	time.Sleep(time.Second * time.Duration(jokeCD))
	jokeLock = false
}

func (j *Joke) TellJoke(client *twitch.Client, msg twitch.PrivateMessage) {
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

	stripped := strings.ReplaceAll(r.Joke, "\n", " ")
	j.TcpChannel <- fmt.Sprintf("tts %s", stripped)

	// Some jokes have \r\n in them - I think we need to filter those out
	jokes := strings.Split(r.Joke, "\n")
	for _, joke := range jokes {
		client.Say(msg.Channel, joke)
	}

}

func (j *Joke) JokeMode(client *twitch.Client, msg twitch.PrivateMessage) {
	j.jokeMode = true
	go func() {
		for {
			select {
			case <-j.jokeModeStop:
				j.jokeMode = false
				return
			default:
				j.TellJoke(client, msg)
				time.Sleep(time.Second * 10)
			}
		}
	}()
}

func (j *Joke) Help() []string {
	return []string{
		"!joke to hear one joke.",
		"!joke mode on|off to enable or disable joke mode",
	}
}
