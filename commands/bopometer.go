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

// here lies the bopometer...
type Bopometer struct {
	Music        *Music
	TCPChannel   chan string
	ratings      map[string]trackInfo // key is spotify track id
	currentTrack trackInfo
	hasBopped    map[string]bool // usernames
	isBopping    bool
}

type trackInfo struct {
	ID      string //spotify track id
	Name    string
	Artists []string
	Rating  float32
}

const (
	boppingWindow    = 25 // seconds
	bopEndWaringTime = 5  // seconds
)

func (b *Bopometer) Init() {
	b.ratings = map[string]trackInfo{}
	b.hasBopped = map[string]bool{}
	j, err := os.ReadFile("./ratings.json")
	if err != nil {
		log.Println("Couldn't load bopometer ratings info from file")
	} else {
		err = json.Unmarshal(j, &b.ratings)
		if err != nil {
			log.Println("Invalid json in bops file")
		}
	}
}

func (b *Bopometer) Run(client *twitch.Client, msg twitch.PrivateMessage) {

	// some one initiates the bopometer by typing the !bop command
	// then for the next n seconds anyone else can !bop to add their "vote"
	// users get one !bop
	// after completion display results and write the current slice to file to persist

	if b.Music.SpotifyClient == nil {
		client.Say(msg.Channel, "Not logged into Spotify. Can't user music commands right now. Tell the streamer to log in and not be a dolt.")
		return
	}

	args := strings.Fields(strings.ToLower(strings.TrimPrefix(msg.Message, "!")))

	if len(args) == 1 {

		if !b.isBopping {
			trackID, isPlaying := b.Music.getCurrentTrackID()
			if !isPlaying {
				client.Say(msg.Channel, "No track is currently playing.")
				return
			}

			// start bopping
			b.isBopping = true
			b.hasBopped[msg.User.Name] = true
			client.Say(msg.Channel, fmt.Sprintf("BOP BOP BOP @%s has started the bopometer! Type !bop to bop", msg.User.DisplayName))
			artists, _ := b.Music.getCurrentTrackArtists()
			song, _ := b.Music.getCurrentTrackTitle()
			b.currentTrack = trackInfo{Name: song, Artists: artists, Rating: 1, ID: trackID}
			b.TCPChannel <- "bop start"
			c := make(chan int)
			go func(chan int) {
				client.Say(msg.Channel, fmt.Sprintf("Bopping has %d seconds left! !bop away!", <-c))
			}(c)
			go func(chan int) {
				bopTimer(c)
				client.Say(msg.Channel, "Bopping has concluded.")
				b.TCPChannel <- "bop stop"
				b.isBopping = false
			}(c)
		} else {
			// already bopping add to the bopping until bopping is complete. bopping
			// if _, ok := b.hasBopped[msg.User.Name]; !ok {
			// 	b.hasBopped[msg.User.Name] = true
			b.currentTrack.Rating++
			// } else {
			// 	client.Say(msg.Channel, fmt.Sprintf("@%s, you've already bopped... stop it.", msg.User.DisplayName))
			// }
		}
		return
	}

	if args[1] == "top" {
		// get top 3 bops
	}
}

func (b *Bopometer) OnUserPart(client *twitch.Client, msg twitch.UserPartMessage) {

}

// search rating by track / artist
// top 3 or 5
// 	- track title/artists and rating

func (b *Bopometer) saveRatingsToFile() {
	json, err := json.Marshal(b.ratings)
	if err != nil {
		log.Println("Couldn't json")
		return
	}
	if err := os.WriteFile("./ratings.json", json, 0644); err != nil {
		log.Println(err.Error())
	}
}

func bopTimer(c chan int) {
	ticker := time.NewTicker(time.Second)
	i := 1
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			i++
			if i == boppingWindow-bopEndWaringTime {
				c <- bopEndWaringTime
			}
			if i >= boppingWindow {
				return
			}
		}
	}
}

func (b *Bopometer) GetBopping() bool {
	return b.isBopping
}

func (b *Bopometer) AddBops(n int) {
	//b.currentTrack.Rating += n
	b.TCPChannel <- fmt.Sprintf("bop add %d", n)
}

func (b *Bopometer) Results(client *twitch.Client, rating string) {
	res, err := strconv.ParseFloat(rating, 32)
	if err != nil {
		log.Println("invalid bop result from overlay", err)
	}
	b.currentTrack.Rating = float32(res)
	if track, exists := b.ratings[b.currentTrack.ID]; exists {
		if b.currentTrack.Rating > track.Rating {
			client.Say("burtstanton", fmt.Sprintf("%s has set a new record with a %.2f rating on the Bopometer!", track.Name, b.currentTrack.Rating))
			b.ratings[b.currentTrack.ID] = b.currentTrack
		}
	} else {
		client.Say("burtstanton", fmt.Sprintf("%s registered a rating of %.2f on the Bopometer!", b.currentTrack.Name, b.currentTrack.Rating))
		b.ratings[b.currentTrack.ID] = b.currentTrack
	}
	b.hasBopped = map[string]bool{}
	b.saveRatingsToFile()
}
