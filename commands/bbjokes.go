package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type Joke struct {
	jokeMode     bool
	jokeModeStop chan bool
}

type apiResponse struct {
	ID     string `json:"id"`
	Joke   string `json:"joke"`
	Status int    `json:"status"`
}

var jokeLock bool = false
var jokeCD int = 5       // seconds
var overloadCD int = 600 // seconds
var canOverload bool = true
var joke *Joke = &Joke{jokeModeStop: make(chan bool)}

func init() {
	RegisterCommand("joke", joke)
}

func (j *Joke) PostInit() {
	j.jokeModeStop = make(chan bool)
	j.jokeMode = false
}

func (j *Joke) Run(msg twitch.PrivateMessage) {
	if jokeLock && !IsMod(msg.User) {
		return
	}
	if !jokeLock {
		jokeLock = true
		go unlockJoke()
	}
	args := strings.Fields(strings.ToLower(strings.TrimPrefix(msg.Message, "!")))
	if len(args) == 1 {
		j.TellJoke(msg, false, false)
		return
	}

	if args[1] == "mode" && IsMod(msg.User) {
		if len(args) < 3 {
			return
		}
		// start
		if args[2] == "start" && !j.jokeMode {
			comm.ToChat(msg.Channel, "Initiating joke mode - prepare for copious amounts of laughter.")
			j.JokeMode(msg)
		}
		// stop
		if args[2] == "stop" && j.jokeMode {
			comm.ToChat(msg.Channel, "Ending joke mode - try to stop laughing now.")
			j.jokeModeStop <- true
		}
		return
	}

	if args[1] == "overload" {
		if !canOverload {
			comm.ToChat(msg.Channel, "I'm all out of jokes... for a little while.")
			return
		}
		canOverload = false
		go func() {
			time.Sleep(time.Second * time.Duration(overloadCD))
			canOverload = true
		}()
		for i := 0; i < 100; i++ {
			j.TellJoke(msg, true, true)
		}
	}
}

func unlockJoke() {
	time.Sleep(time.Second * time.Duration(jokeCD))
	jokeLock = false
}

func (j *Joke) TellJoke(msg twitch.PrivateMessage, voiceOnly, randomVoice bool) {
	// Fetch a joke from icanhazdadjoke api
	req, err := http.NewRequest("GET", "https://icanhazdadjoke.com", nil)
	if err != nil {
		comm.ToChat(msg.Channel, "Sorry, couldn't connect to joke dispensory")
		log.Println("Couldn't access joke api: ", err.Error())
		return
	}
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	//resp, err := http.Get("https://icanhazdadjoke.com/")
	if err != nil {
		comm.ToChat(msg.Channel, "Sorry, couldn't connect to joke dispensory")
		log.Println("Couldn't access joke api: ", err.Error())
		return
	}
	r := apiResponse{}
	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		comm.ToChat(msg.Channel, "I dropped the joke on the way back from the store, sorry.")
		log.Println(err.Error())
		return
	}

	stripped := strings.ReplaceAll(r.Joke, "\n", " ")
	if comm.IsConnectedToOverlay() {
		comm.ToOverlay(fmt.Sprintf("tts true %t %s", randomVoice, stripped))
	}
	// Some jokes have \r\n in them - I think we need to filter those out
	if voiceOnly {
		return
	}
	jokes := strings.Split(r.Joke, "\n")
	for _, joke := range jokes {
		comm.ToChat(msg.Channel, joke)
	}

}

func (j *Joke) JokeMode(msg twitch.PrivateMessage) {
	j.jokeMode = true
	go func() {
		for {
			select {
			case <-j.jokeModeStop:
				j.jokeMode = false
				return
			default:
				j.TellJoke(msg, true, true)
				time.Sleep(time.Second * 10)
			}
		}
	}()
}

func (j *Joke) Help() []string {
	return []string{
		"!joke to hear one joke.",
		"!joke mode start|stop to enable or disable joke mode",
		"!joke overload - don't do this. please.",
	}
}
