package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gempir/go-twitch-irc/v2"
	"github.com/zmb3/spotify"
)

type Music struct {
	SpotifyClient *spotify.Client
	Tokens        map[string]int
	persist       bool
}

const skipCost = 5       //tokens
const previousCost = 100 //tokens

var spotifyAuth = spotify.NewAuthenticator("http://localhost:8079/spotify_authcb",
	spotify.ScopeUserReadPrivate,
	spotify.ScopeUserReadCurrentlyPlaying,
	spotify.ScopeUserReadRecentlyPlayed,
	spotify.ScopeUserModifyPlaybackState)
var spotifyAuthCh = make(chan *spotify.Client)
var spotifyState = "test123"

func (m *Music) Init() {

	http.HandleFunc("/spotify_authcb", completeAuth)
	go http.ListenAndServe(":8079", nil)

	url := spotifyAuth.AuthURL(spotifyState)
	fmt.Println("Auth url for spotify: ", url)

	m.SpotifyClient = <-spotifyAuthCh

	user, err := m.SpotifyClient.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Logged in to Spotify as: ", user.ID)

	// tokens init
	m.Tokens = make(map[string]int)
	m.persist = true
	j, err := os.ReadFile("./tokens.json")
	if err != nil {
		log.Println("Couldn't load token info from file")
		m.persist = false
	} else {
		err = json.Unmarshal(j, &m.Tokens)
		if err != nil {
			log.Println("Invalid json in tokens file")
			m.persist = false
		}
	}
}

func (m *Music) Run(client *twitch.Client, msg twitch.PrivateMessage) {

	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))

	if len(args) == 1 {
		client.Say(msg.Channel, "Use '!music current' to get the currently playing track or '!music last' to get the last track played")
		return
	}

	if args[1] == "current" {
		cp, err := m.SpotifyClient.PlayerCurrentlyPlaying()
		if !cp.Playing {
			client.Say(msg.Channel, "Not playing music right now")
			return
		}
		if err != nil {
			log.Println("SpotifyClient err: ", err.Error())
			client.Say(msg.Channel, "Could not get playback data atm.")
			return
		}
		artists := ""
		for k, v := range cp.Item.Artists {
			artists += v.Name
			if k != len(cp.Item.Artists)-1 {
				artists += ", "
			}
		}
		retMsg := fmt.Sprintf(`Current Song: "%s" by %s. Listen on Spotify: %s`, cp.Item.Name, artists, cp.Item.ExternalURLs["spotify"])
		client.Say(msg.Channel, retMsg)
		return
	}

	if args[1] == "last" {
		rp, err := m.SpotifyClient.PlayerRecentlyPlayed()
		if err != nil {
			log.Println("SpotifyClient err: ", err.Error())
			client.Say(msg.Channel, "Could not get playback data atm.")
			return
		}
		last := rp[0].Track
		artists := ""
		for k, v := range last.Artists {
			artists += v.Name
			if k != len(last.Artists)-1 {
				artists += ", "
			}
		}
		retMsg := fmt.Sprintf(`Last Song: "%s" by %s. Listen on Spotify: %s`, last.Name, artists, last.ExternalURLs["spotify"])
		client.Say(msg.Channel, retMsg)
		return
	}

	if args[1] == "grant" {
		if !isMod(msg.User) || len(args) < 3 {
			return
		}
		var numberTokens int
		if n, err := strconv.Atoi(args[3]); err != nil {
			numberTokens = 1
		} else {
			numberTokens = n
		}
		// no validation for twitch users here - but we will save and fetch them in all lowercase
		username := strings.ToLower(args[2])
		m.grantToken(username, numberTokens)
		return
	}

	if args[1] == "request" {
		if len(args) < 3 {
			return
		}
		split := strings.Split(args[2], "/")
		if len(split) != 5 {
			client.Say(msg.Channel, "Looks like an invalid request link")
			return
		}
		var sid spotify.ID = spotify.ID(strings.Split(split[4], "?")[0])
		_, message := m.request(msg.User, sid) // dumping status check for now since all roads lead to rome
		client.Say(msg.Channel, message)
	}

	if args[1] == "skip" {
		if m.getTokenCount(msg.User) < skipCost {
			client.Say(msg.Channel, fmt.Sprintf("@%s you don't have enough tokens to skip this song. Deal with it.", msg.User.DisplayName))
			return
		}

		err := m.SpotifyClient.Next()
		if err != nil {
			log.Println("Could not skip track - ", err.Error())
			client.Say(msg.Channel, "Sorry I failed to skip the track, I won't take your tokens. Though I could. If I wanted to.")
			return
		}

		m.setTokenCount(msg.User.DisplayName, m.getTokenCount(msg.User)-skipCost)
		plural := ""
		if m.getTokenCount(msg.User) > 1 {
			plural = "s"
		}
		client.Say(msg.Channel, fmt.Sprintf("Are you happy @%s? You skipped everyone's favorite song...", msg.User.DisplayName))
		if m.getTokenCount(msg.User) > 0 {
			client.Say(msg.Channel, fmt.Sprintf("@%s, also, you only have %d token%s left", msg.User.DisplayName, m.getTokenCount(msg.User), plural))
		} else {
			client.Say(msg.Channel, fmt.Sprintf("@%s, also, you have no tokens left. Sad.", msg.User.DisplayName))
		}

		return
	}

	if args[1] == "previous" {
		if m.getTokenCount(msg.User) < previousCost {
			client.Say(msg.Channel, fmt.Sprintf("@%s you don't have enough tokens to return to the past.", msg.User.DisplayName))
			return
		}

		err := m.SpotifyClient.Previous()
		if err != nil {
			log.Println("Could not go to previous track - ", err.Error())
			client.Say(msg.Channel, "Couldn't go back to the previous track. Maybe it never existesd. For all we know the universe started last Thursday.")
			return
		}

		m.setTokenCount(msg.User.DisplayName, m.getTokenCount(msg.User)-previousCost)
		plural := ""
		if m.getTokenCount(msg.User) > 1 {
			plural = "s"
		}
		client.Say(msg.Channel, fmt.Sprintf("Okay @%s, I guess we have to go back to the last song.", msg.User.DisplayName))
		if m.getTokenCount(msg.User) > 0 {
			client.Say(msg.Channel, fmt.Sprintf("@%s, also, you only have %d token%s left", msg.User.DisplayName, m.getTokenCount(msg.User), plural))
		} else {
			client.Say(msg.Channel, fmt.Sprintf("@%s, also, you have no tokens left. Sad.", msg.User.DisplayName))
		}
		return
	}
}

func (m Music) grantToken(username string, number int) {
	m.Tokens[username] += number
	if m.persist {
		m.saveTokensToFile()
	}
}

// Put in a request for the music player from the given user for the given song link
func (m Music) request(user twitch.User, song spotify.ID) (bool, string) {
	if m.getTokenCount(user) <= 0 {
		return false, fmt.Sprintf("@%s you need a token to make a request. Get tokens from the token machine.", user.Name)
	}

	err := m.SpotifyClient.QueueSong(song)
	if err != nil {
		return false, fmt.Sprintf("There was an error queing the song - may be an invalid track id")
	}

	m.Tokens[strings.ToLower(user.Name)]--
	if m.persist {
		m.saveTokensToFile()
	}

	trackInfo, err := m.SpotifyClient.GetTrack(song)
	if err != nil {
		// can't get track info for whatever reason,
		// but did get queued
		return true, "Track queued successfully"
	}

	artists := ""
	for k, v := range trackInfo.SimpleTrack.Artists {
		artists += v.Name
		if k != len(trackInfo.SimpleTrack.Artists)-1 {
			artists += ", "
		}
	}
	return true, fmt.Sprintf(`Added "%s" by %s to the queue.`, trackInfo.SimpleTrack.Name, artists)
}

// Get a user's current token count
func (m Music) getTokenCount(user twitch.User) int {
	username := strings.ToLower(user.Name)
	// No one gets any tokens!!!!
	return m.Tokens[username]
}

func (m *Music) setTokenCount(userName string, number int) {
	m.Tokens[userName] = number
	if m.persist {
		m.saveTokensToFile()
	}
}

func (m Music) saveTokensToFile() {
	json, err := json.Marshal(m.Tokens)
	if err != nil {
		log.Println("Couldn't json")
		return
	}
	if err := os.WriteFile("./tokens.json", json, 0644); err != nil {
		log.Println(err.Error())
	}
}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := spotifyAuth.Token(spotifyState, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != spotifyState {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, spotifyState)
	}
	// use the token to get an authenticated client
	client := spotifyAuth.NewClient(tok)
	fmt.Fprintf(w, "Login completed!")
	spotifyAuthCh <- &client
}
