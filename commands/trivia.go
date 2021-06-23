package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"strconv"
	"errors"
	"log"
	"time"
	"context"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

const (
	
	rcSuccess = iota
	rcNoResults
	rcInvalidParameter
	rcTokenNotFound
	rcTokenEmpty
	
	triviaApiEndpoint = "https://opentdb.com/api.php"
	
	roundTime  = 30 // seconds
	numberOfRounds = 10
)

var triviaCancel context.CancelFunc

type triviaPlayer struct {
	user  twitch.User
	score int
}

type triviaAPIResponse struct {
	ResponseCode int `json:"response_code"`
	Questions []question `json:"results"`
}
type question struct {
	Category         string
	Type             string
	Difficulty       string
	Text             string   `json:"question"`
	CorrectAnswer    string   `json:"correct_answer"`
	IncorrectAnswers []string `json:"incorrect_answers"`
}

type category struct {
	ID   int
	Name string
}

type Trivia struct {
	players     map[string]*triviaPlayer
	roundNumber int
	questions   []question

	AnswerChannel chan twitch.PrivateMessage
}

func NewTrivia() *Trivia {
	return &Trivia{
		players: map[string]*triviaPlayer{},
		roundNumber: 0,
		questions: []question{},
		AnswerChannel: make(chan twitch.PrivateMessage),
	}
}

func (t *Trivia) Run(client *twitch.Client, msg twitch.PrivateMessage) {
	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))
	if len(args) < 2 {
		return
	}

	if args[1] == "categories" {
		categories, err := getCategories()
		if err != nil {
			comm.ToChat(msg.Channel, "Couldn't get the trivia categories... Sorry.")
			return
		}
		for _, category := range categories {
			comm.ToChat(msg.Channel, fmt.Sprintf("%d. %s", category.ID, category.Name))
		}
		return
	}

	// start
	//	!trivia start
	if args[1] == "start" && len(args) >= 3 {
		catNumber, err := strconv.Atoi(args[2])
		if err != nil {
			return
		}
		// get the questions from the api
		err = t.getQuestions(catNumber)
		if err != nil {
			comm.ToChat(msg.Channel, "I couldn't get any trivia questions. Everyone knows everything anyways.")
			log.Println(err.Error())
			return
		}

		// start the game
		var ctx context.Context
		ctx, triviaCancel = context.WithCancel(context.Background())
		go t.run(ctx, client, msg.Channel)	
	}
	
	// stop
	if args[1] == "stop" && IsMod(msg.User) && triviaCancel != nil {
		comm.ToChat(msg.Channel, fmt.Sprintf("Stopping trivia because @%s hates fun", msg.User.DisplayName))
		triviaCancel()
		triviaCancel = nil
	}
	// reset
}

func (t *Trivia) Init() {

}

func (t *Trivia) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {

}

func (t *Trivia) Help() []string {
	return []string{
		"!trivia start [category number] to start a game of trivia",
		"!trivia categories to see a list of categories",
	}
}

func (t *Trivia) getQuestions(category int) error {
	// get questions from api based on category
	url := fmt.Sprintf("%s?amount=%d&category=%d", triviaApiEndpoint, numberOfRounds, category)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	responseData := triviaAPIResponse{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&responseData)
	if err != nil {
		return err
	}
	// check the response code from the api
	if responseData.ResponseCode != rcSuccess {
		return errors.New("bad request to trivia api")
	}
	t.questions = responseData.Questions
	return nil
}

func (t *Trivia) run(ctx context.Context, client *twitch.Client, channel string) {
	// ask a question
	comm.ToChat(channel, t.questions[t.roundNumber].Text)
	// wait for answer
	ticker := time.NewTicker(time.Second)
	tick := 0
	defer ticker.Stop()
	Loop:
	for {
		select {
			case <-ctx.Done():
				return
			case msg := <-t.AnswerChannel:
				if msg.Message != t.questions[t.roundNumber].Text {
					continue
				}
				// correct answer
				if _, ok := t.players[msg.User.ID]; !ok {
					t.players[msg.User.ID] = &triviaPlayer{
						user: msg.User,
						score: 0,
					}
				}
				t.players[msg.User.ID].score += 1
				break Loop
			case <-ticker.C:
				tick++
				if tick >= roundTime {
					comm.ToChat(channel, "times up, next question")
					break Loop
				}
		}	
	}
	// check if there are more questions
	t.roundNumber++
	if t.roundNumber >= numberOfRounds {
		// do end of game things here
		return
	}
	// goto start
	// ctx, cancel := context.WithCancel(context.Background())
	t.run(ctx, client, channel)
}

func (t *Trivia) Reset() {
	t.roundNumber = 0
	t.players = map[string]*triviaPlayer{}
	t.questions = []question{}
}

func getCategories() ([]category, error) {
	// get list of categories from api
	categories := []category{}

	resp, err := http.Get("https://opentdb.com/api_category.php")
	if err != nil {
		return categories, err
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	rdata := struct {
		Categories []category `json:"trivia_categories"`
	}{}
	err = dec.Decode(&rdata)
	if err != nil {
		return categories, err
	}

	return rdata.Categories, nil
}
