package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/MattSwanson/burtbot/console"
	"github.com/MattSwanson/burtbot/web"
	"github.com/gempir/go-twitch-irc/v2"
	"github.com/zmb3/spotify"
)

type Music struct {
	SpotifyClient *spotify.Client
}

const skipCost = 5       //tokens
const previousCost = 100 //tokens
const requestQueueURL = "https://burtbot.app/request_queue"
const requestHistorySize = 5

var spotifyAuth = spotify.NewAuthenticator("https://burtbot.app/spotify_authcb",
	spotify.ScopeUserReadPrivate,
	spotify.ScopeUserReadCurrentlyPlaying,
	spotify.ScopeUserReadRecentlyPlayed,
	spotify.ScopeUserReadPlaybackState,
	spotify.ScopeUserModifyPlaybackState)
var spotifyAuthCh = make(chan *spotify.Client)
var spotifyState = "test123"
var mu *Music = &Music{}
var nowPlaying string
var songRequestQueueTpl *template.Template
var youtubeAPIURL = "https://www.googleapis.com/youtube/v3/videos?key=%s&part=snippet&part=contentDetails&id=%s"
var isAcceptingRequests bool
var nowPlayingRequest *SongRequest

type SongRequest struct {
	SongTitle   string
	SongArtists []string
	SongLink    string
	Service     string
	User        string
	Added       time.Time
	Duration    int
}

type YoutubeVideoInfo struct {
	Snippet struct {
		Title      string `json:"title"`
		Author     string `json:"channelTitle"`
		Thumbnails struct {
			Default struct {
				Height int    `json:"height"`
				Width  int    `json:"width"`
				URL    string `json:"url"`
			}
		}
	}
	ContentDetails struct {
		Duration string `json:"duration"`
	}
}

type YoutubeAPIResponse struct {
	Items []YoutubeVideoInfo
}

var requestQueue []*SongRequest = []*SongRequest{}
var lastRequests []*SongRequest = []*SongRequest{}

func init() {
	songRequestQueueTpl = template.Must(template.ParseFiles("templates/song_request_queue.gohtml"))
	web.AuthHandleFunc("/request_queue", showSongRequestQueue)
	web.AuthHandleFunc("/remove_request", removeRequest)
	web.AuthHandleFunc("/play_request", setRequestPlaying)
	RegisterCommand("music", mu)
}

func (m *Music) PostInit() {
	go func() {
		http.HandleFunc("/spotify_authcb", completeAuth)
		m.SpotifyClient = <-spotifyAuthCh
		console.SetSpotifyStatus(true)
		go func() {
			for {
				playerState, err := m.SpotifyClient.PlayerState()
				// Need to check if the current track is nil also
				if err != nil || playerState.CurrentlyPlaying.Item == nil {
					time.Sleep(2000 * time.Millisecond)
					continue
				}
				if !playerState.CurrentlyPlaying.Playing {
					if nowPlaying != "" {
						nowPlaying = ""
						comm.ToOverlay("nowplaying off")
					}
				} else {
					track, playing := m.getCurrentTrackTitle()
					if playing && track != "" {
						artists, _ := m.getCurrentTrackArtists()
						track = fmt.Sprintf("%s - %s", track, strings.Join(artists, ", "))
						if track != nowPlaying {
							nowPlaying = track
							comm.ToOverlay(fmt.Sprintf("nowplaying %s", nowPlaying))
						}
					}
				}
				time.Sleep(2000 * time.Millisecond)
			}
		}()
	}()
	loadQueueFromFile()
	mu = m
}

func (m *Music) Run(msg twitch.PrivateMessage) {

	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))

	if len(args) == 1 {
		comm.ToChat(msg.Channel, "Use '!music current' to get the currently playing track or '!music last' to get the last track played")
		return
	}

	if m.SpotifyClient == nil {
		comm.ToChat(msg.Channel, "Not logged into Spotify. Can't user music commands right now. Tell the streamer to log in and not be a dolt.")
		return
	}

	if args[1] == "current" {
		/*	cp, err := m.SpotifyClient.PlayerCurrentlyPlaying()
			if !cp.Playing {
				comm.ToChat(msg.Channel, "Not playing music right now")
				return
			}
			if err != nil {
				log.Println("SpotifyClient err: ", err.Error())
				comm.ToChat(msg.Channel, "Could not get playback data atm.")
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
			comm.ToChat(msg.Channel, retMsg)
			return
		*/
		if nowPlayingRequest == nil {
			comm.ToChat(msg.Channel, "There is no song currently playing.")
			return
		}

		artists := strings.Join(nowPlayingRequest.SongArtists, ",")
		text := fmt.Sprintf(`Current Song: "%s" by %s. As requested by %s - Link: %s`, nowPlayingRequest.SongTitle, artists, nowPlayingRequest.User, nowPlayingRequest.SongLink)
		comm.ToChat(msg.Channel, text)
		return
	}

	if args[1] == "last" {
		/*
			rp, err := m.SpotifyClient.PlayerRecentlyPlayed()
			if err != nil {
				log.Println("SpotifyClient err: ", err.Error())
				comm.ToChat(msg.Channel, "Could not get playback data atm.")
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
			comm.ToChat(msg.Channel, retMsg)
			return
		*/
		if len(lastRequests) == 0 {
			comm.ToChat(msg.Channel, "There is no request history.")
			return
		}

		lastSong := lastRequests[0]
		artists := strings.Join(lastSong.SongArtists, ",")
		text := fmt.Sprintf(`Last Song: "%s" by %s. As Requested by %s - Link: %s`, lastSong.SongTitle, artists, lastSong.User, lastSong.SongLink)
		comm.ToChat(msg.Channel, text)
		return
	}

	if args[1] == "request" {

		if args[2] == "enable" && IsMod(msg.User) {
			isAcceptingRequests = true
			comm.ToChat(msg.Channel, "Song requests have been enabled. Use !sr <song link> to request a song.")
			return
		}

		if args[2] == "disable" && IsMod(msg.User) {
			isAcceptingRequests = false
			comm.ToChat(msg.Channel, "The song request queue has been closed. Go home.")
			return
		}

		if !isAcceptingRequests {
			comm.ToChat(msg.Channel, "Sorry, requests are not enabled at the moment. Please try again some other century when requests are actually enabled. Thanks for trying though. Your effort is appreciated.")
			return
		}

		if len(args) < 3 {
			return
		}

		req, err := processRequest(args[2], msg.User.DisplayName)
		if err != nil {
			comm.ToChat(msg.Channel, "Error adding request to the queue, please try again later.")
			return
		}
		duration := 0
		for _, request := range requestQueue {
			duration += request.Duration
		}
		duration += 60_000 - (duration % 60_000)
		duration /= 60_000
		mins := duration % 60
		hours := duration / 60
		timeToPlay := fmt.Sprintf("%dmin", mins)
		if hours > 0 {
			timeToPlay = fmt.Sprintf("%dhr %s", hours, timeToPlay)
		}
		requestQueue = append(requestQueue, req)
		comm.ToChat(msg.Channel, fmt.Sprintf("@%s: %s - %s has been added at #%d in the queue. Played in approx. %s", msg.User.DisplayName, req.SongTitle, strings.Join(req.SongArtists, ","), len(requestQueue), timeToPlay))
		saveQueueToFile()
	}

	if args[1] == "skip" {
		/*
			if GetTokenCount(msg.User).Cmp(big.NewInt(skipCost)) == -1 {
				comm.ToChat(msg.Channel, fmt.Sprintf("@%s you don't have enough tokens to skip this song. Deal with it.", msg.User.DisplayName))
				return
			}

			err := m.SpotifyClient.Next()
			if err != nil {
				log.Println("Could not skip track - ", err.Error())
				comm.ToChat(msg.Channel, "Sorry I failed to skip the track, I won't take your tokens. Though I could. If I wanted to.")
				return
			}

			DeductTokens(msg.User.DisplayName, big.NewInt(skipCost))
			tokensLeft := GetTokenCount(msg.User)
			plural := ""
			if tokensLeft.Cmp(big.NewInt(1)) == 1 {
				plural = "s"
			}
			comm.ToChat(msg.Channel, fmt.Sprintf("Are you happy @%s? You skipped everyone's favorite song...", msg.User.DisplayName))
			if tokensLeft.Cmp(big.NewInt(0)) == 1 {
				comm.ToChat(msg.Channel, fmt.Sprintf("@%s, also, you only have %d token%s left", msg.User.DisplayName, GetTokenCount(msg.User), plural))
			} else {
				comm.ToChat(msg.Channel, fmt.Sprintf("@%s, also, you have no tokens left. Sad.", msg.User.DisplayName))
			}
		*/
		comm.ToChat(msg.Channel, "Song skipping has been disabled for the time being. Good job for knowing this command existed in the first place.")

		return
	}

	if args[1] == "previous" {
		/*
			if GetTokenCount(msg.User).Cmp(big.NewInt(previousCost)) == -1 {
				comm.ToChat(msg.Channel, fmt.Sprintf("@%s you don't have enough tokens to return to the past.", msg.User.DisplayName))
				return
			}

			err := m.SpotifyClient.Previous()
			if err != nil {
				log.Println("Could not go to previous track - ", err.Error())
				comm.ToChat(msg.Channel, "Couldn't go back to the previous track. Maybe it never existesd. For all we know the universe started last Thursday.")
				return
			}

			DeductTokens(msg.User.DisplayName, big.NewInt(previousCost))
			tokensLeft := GetTokenCount(msg.User)
			plural := ""
			if tokensLeft.Cmp(big.NewInt(1)) == 1 {
				plural = "s"
			}
			comm.ToChat(msg.Channel, fmt.Sprintf("Okay @%s, I guess we have to go back to the last song.", msg.User.DisplayName))
			if tokensLeft.Cmp(big.NewInt(0)) == 1 {
				comm.ToChat(msg.Channel, fmt.Sprintf("@%s, also, you only have %d token%s left", msg.User.DisplayName, GetTokenCount(msg.User), plural))
			} else {
				comm.ToChat(msg.Channel, fmt.Sprintf("@%s, also, you have no tokens left. Sad.", msg.User.DisplayName))
			}
		*/
		comm.ToChat(msg.Channel, "We can not return to the past.")
		return
	}

	if args[1] == "nptext" && IsMod(msg.User) {
		if len(args) < 3 {
			return
		}

		if args[2] == "top" || args[2] == "bottom" {
			comm.ToOverlay(fmt.Sprintf("nptext %s", args[2]))
		}

		if args[2] == "off" {
			comm.ToOverlay("nowplaying off")
		}
	}
}

func (m *Music) getCurrentTrackTitle() (string, bool) {
	cp, err := m.SpotifyClient.PlayerCurrentlyPlaying()
	if err != nil {
		return "", false
	}
	trackTitle := ""
	if cp != nil {
		trackTitle = cp.Item.Name
	}
	return trackTitle, true
}

func GetCurrentTrackTitle() (string, bool) {
	return mu.getCurrentTrackTitle()
}

func (m *Music) getCurrentTrackArtists() ([]string, bool) {
	cp, err := m.SpotifyClient.PlayerCurrentlyPlaying()
	if err != nil || cp.Item == nil {
		return []string{}, false
	}
	artists := []string{}
	for _, artist := range cp.Item.Artists {
		artists = append(artists, artist.Name)
	}
	return artists, true
}

func GetCurrentTrackArtists() ([]string, bool) {
	return mu.getCurrentTrackArtists()
}

func (m *Music) getCurrentTrackID() (string, bool) {
	cp, err := m.SpotifyClient.PlayerCurrentlyPlaying()
	if err != nil || cp.Item == nil {
		return "", false
	}
	return string(cp.Item.ID.String()), true
}

func GetCurrentTrackID() (string, bool) {
	return mu.getCurrentTrackID()
}

func processRequest(link string, user string) (*SongRequest, error) {

	service := "Unknown"
	songTitle := "Unknown"
	songArtists := []string{"Unknown"}
	duration := 0
	if strings.Contains(link, "open.spotify.com") {
		service = "Spotify"
		spotifyID := extractSpotifyIDFromLink(link)
		info, err := mu.SpotifyClient.GetTrack(spotify.ID(spotifyID))
		if err != nil {
			log.Println("Couldn't get spotify track info: ", err)
			return nil, err
		}
		songTitle = info.Name
		songArtists = []string{}
		for _, artist := range info.Artists {
			songArtists = append(songArtists, artist.Name)
		}
		duration = info.Duration
	} else if strings.Contains(link, "youtube.com") || strings.Contains(link, "youtu.be") {
		service = "Youtube"
		videoInfo, err := getYoutubeVideoInfoFromLink(link)
		if err != nil {
			log.Println("Couldn't get video info from youtube link: ", err)
			return nil, err
		}
		songTitle = videoInfo.Snippet.Title
		songArtists = []string{videoInfo.Snippet.Author}
		duration = youtubeVideoDurationToMS(videoInfo.ContentDetails.Duration)
	}
	sr := SongRequest{
		SongTitle:   songTitle,
		SongArtists: songArtists,
		SongLink:    link,
		User:        user,
		Service:     service,
		Added:       time.Now(),
		Duration:    duration,
	}
	return &sr, nil
}

func extractSpotifyIDFromLink(link string) string {
	split := strings.Split(link, "/")
	split = strings.Split(split[4], "?")
	return split[0]
}

func extractYoutubeVideoIDFromLink(link string) string {
	// depending on the link used the id could be in multiple places...
	parsed, err := url.Parse(link)
	if err != nil {
		return ""
	}
	if parsed.Path == "/watch" {
		q, err := url.ParseQuery(parsed.RawQuery)
		if err != nil {
			return ""
		}
		return q["v"][0]
	}
	if strings.HasPrefix(parsed.Path, "/live") {
		split := strings.Split(parsed.Path, "/")
		if len(split) < 2 {
			return ""
		}
		return split[1]
	}
	return strings.TrimPrefix(parsed.Path, "/")
}

func youtubeVideoDurationToMS(duration string) int {
	ms := 0
	duration = strings.TrimPrefix(duration, "PT")
	spl := strings.Split(duration, "H")
	if len(spl) != 1 {
		h, err := strconv.Atoi(spl[0])
		if err != nil {
			return 0
		}
		ms = h * 60 * 60 * 1000
		duration = strings.Join(spl[1:len(spl)], "")
	}
	spl = strings.Split(duration, "M")
	if len(spl) != 1 {
		m, err := strconv.Atoi(spl[0])
		if err != nil {
			return 0
		}
		ms = ms + m*60*1000
		duration = strings.Join(spl[1:len(spl)], "")
	}
	spl = strings.Split(duration, "S")
	if len(spl) != 1 {
		s, err := strconv.Atoi(spl[0])
		if err != nil {
			return 0
		}
		ms = ms + s*1000
		duration = strings.Join(spl[1:len(spl)], "")
	}
	return ms
}

func getYoutubeVideoInfoFromLink(link string) (*YoutubeVideoInfo, error) {
	videoID := extractYoutubeVideoIDFromLink(link)
	reqUrl := fmt.Sprintf(youtubeAPIURL, os.Getenv("YOUTUBE_API_KEY"), videoID)
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	respStruct := struct {
		Items []YoutubeVideoInfo
	}{}

	err = json.NewDecoder(resp.Body).Decode(&respStruct)
	if err != nil {
		return nil, err
	}
	if len(respStruct.Items) == 0 {
		return nil, errors.New("Received blank response from youtube. So sad. But such is life")
	}
	return &respStruct.Items[0], nil
}

// Put in a request for the music player from the given user for the given song link
func (m Music) request(user twitch.User, song spotify.ID) (bool, string) {
	numTokens := GetTokenCount(user)
	if numTokens.Cmp(big.NewInt(0)) == -1 || numTokens.Cmp(big.NewInt(0)) == 0 {
		return false, fmt.Sprintf("@%s you need a token to make a request. Get tokens from the token machine.", user.Name)
	}

	err := m.SpotifyClient.QueueSong(song)
	if err != nil {
		return false, "There was an error queing the song - may be an invalid track id"
	}

	DeductTokens(user.DisplayName, big.NewInt(1))

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
	spotifyAuthCh <- &client
	http.Redirect(w, r, "https://burtbot.app/services_auth", http.StatusSeeOther)
}

func GetSpotifyAuthStatus() bool {
	return mu.SpotifyClient != nil
}

func GetSpotifyLink() string {
	url := spotifyAuth.AuthURL(spotifyState)
	return url
}

func (m Music) Help() []string {
	return []string{
		"This all assumes music is playing and someone remember to log in...",
		"!music current will show the currently playing song",
		"!music last will show the last song played",
		"!music request [song link] will add a song to the queue for 1 token",
	}
}

func loadQueueFromFile() {
	j, err := os.ReadFile("./requestQueue.json")
	if err != nil {
		log.Println("Couldn't load request queue from file ", err)
		return
	}
	if err := json.Unmarshal(j, &requestQueue); err != nil {
		log.Println("Couldn't unmarshal queue from file ", err)
	}
}

func saveQueueToFile() {
	json, err := json.Marshal(requestQueue)
	if err != nil {
		log.Println("Couldn't marshal request queue into JSON ", err)
		return
	}
	if err := os.WriteFile("./requestQueue.json", json, 0644); err != nil {
		log.Println(err.Error())
	}
}

func IsLoggedInToSpotify() bool {
	return mu.SpotifyClient != nil
}

func removeRequestFromQueue(id int) {
	// remove the item from the request queue at index [id]
	if id == len(requestQueue)-1 {
		requestQueue = requestQueue[0 : len(requestQueue)-1]
	} else {
		requestQueue = append(requestQueue[0:id], requestQueue[id+1:len(requestQueue)]...)
	}
	saveQueueToFile()
}

func showSongRequestQueue(w http.ResponseWriter, r *http.Request) {
	nowPlaying := &SongRequest{}
	if nowPlayingRequest != nil {
		nowPlaying = nowPlayingRequest
	}
	d := struct {
		CurrentQueue []*SongRequest
		NowPlaying   *SongRequest
		History      []*SongRequest
	}{
		CurrentQueue: requestQueue,
		NowPlaying:   nowPlaying,
		History:      lastRequests,
	}
	err := songRequestQueueTpl.ExecuteTemplate(w, "song_request_queue.gohtml", d)
	if err != nil {
		fmt.Fprint(w, err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func setRequestPlaying(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil || id > len(requestQueue)-1 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if nowPlayingRequest != nil {
		lastRequests = append([]*SongRequest{nowPlayingRequest}, lastRequests...)
		if len(lastRequests) > requestHistorySize {
			lastRequests = lastRequests[:requestHistorySize]
		}
	}
	nowPlayingRequest = requestQueue[id]
	removeRequestFromQueue(id)
	if nowPlayingRequest.Service == "Youtube" {
		comm.ToOverlay(fmt.Sprintf("nowplaying %s - %s", nowPlayingRequest.SongTitle, nowPlayingRequest.SongArtists[0]))
	}
	http.Redirect(w, r, requestQueueURL, http.StatusSeeOther)
}

func removeRequest(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil || id > len(requestQueue)-1 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	removeRequestFromQueue(id)
	http.Redirect(w, r, requestQueueURL, http.StatusSeeOther)
}
