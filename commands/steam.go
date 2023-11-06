package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type Steam struct{}

type appEntry struct {
	AppID           int    `json:"appid"`
	PlaytimeForever int    `json:"playtime_forever"`
	TimeLastPlayed  int    `json:"rtime_last_played"`
	Name            string `json:"name"`
	ImgIconURL      string `json:"img_icon_url"`
}

type steamApiResponse struct {
	Response struct {
		Games []appEntry `json:"games"`
	} `json:"response"`
}

var steam *Steam = &Steam{}
var userID string = "76561197968481769"

func init() {
	RegisterCommand("steam", steam)
}

func (s *Steam) PostInit() {

}

func (s *Steam) Run(msg twitch.PrivateMessage) {
	if !IsMod(msg.User) {
		return
	}

	args := strings.Fields(strings.ToLower(strings.TrimPrefix(msg.Message, "!")))
	if len(args) < 2 {
		return
	}

	if args[1] == "random" {
		if comm.IsConnectedToOverlay() {
			comm.ToOverlay("steam")
			return
		}
		gameToPlay := getRandomGame(msg.Channel)
		comm.ToChat(msg.Channel, fmt.Sprintf("You shall play %s.", gameToPlay))
	}
}

//TODO Return an error instead of handling here
//TODO return an appEntry object instead of string name so we can send the img url to the overlay
func getRandomGame(channel string) string {

	apiKey := os.Getenv("STEAM_API_KEY")
	url := fmt.Sprintf("http://api.steampowered.com/IPlayerService/GetOwnedGames/v0001/?key=%s&steamid=%s&format=json&include_appinfo=1", apiKey, userID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		comm.ToChat(channel, "Can't make request to Steam API at the moment")
		log.Println("Steam api error: ", err.Error())
		return "make an error"
	}
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		comm.ToChat(channel, "Error making request to Steam API")
		log.Println("Error accessing Steam API ", err.Error())
		return "make an error"
	}
	r := steamApiResponse{}
	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		comm.ToChat(channel, "Error decoding JSON from Steam API")
		log.Println(err.Error())
		return "make an error"
	}

	filtered := []string{}
	for _, game := range r.Response.Games {
		if game.PlaytimeForever < 30 {
			filtered = append(filtered, game.Name)
		}
	}

	randApp := filtered[rand.Intn(len(filtered))]
	return randApp
}

func (s *Steam) Help() []string {
	return []string{"Todo: Do this"}
}
