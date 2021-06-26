package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
	"sort"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

// here lies the bopometer...
type Bopometer struct {
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

type tracks []trackInfo
func (t tracks) Len() int { return len(t) }
func (t tracks) Less(i, j int) bool { return t[i].Rating > t[j].Rating }
func (t tracks) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

const (
	boppingWindow    = 25 // seconds
	bopEndWaringTime = 5  // seconds
)

var bopometer *Bopometer = &Bopometer{}

func init() {
	bopometer.ratings = map[string]trackInfo{}
	bopometer.hasBopped = map[string]bool{}
	j, err := os.ReadFile("./ratings.json")
	if err != nil {
		log.Println("Couldn't load bopometer ratings info from file")
	} else {
		err = json.Unmarshal(j, &bopometer.ratings)
		if err != nil {
			log.Println("Invalid json in bops file")
		}
	}
	comm.SubscribeToReply("bop", bopometer.Results)
	SubscribeToRawMsg(bopometer.handleRawMessage)
	RegisterCommand("bop", bopometer)
}

func (b *Bopometer) PostInit() {

}

func (b *Bopometer) Run(msg twitch.PrivateMessage) {

	// some one initiates the bopometer by typing the !bop command
	// then for the next n seconds anyone else can !bop to add their "vote"
	// users get one !bop
	// after completion display results and write the current slice to file to persist

	if !IsLoggedInToSpotify() {
		comm.ToChat(msg.Channel, "Not logged into Spotify. Can't user music commands right now. Tell the streamer to log in and not be a dolt.")
		return
	}

	args := strings.Fields(strings.ToLower(strings.TrimPrefix(msg.Message, "!")))

	if len(args) == 1 {

		if !b.isBopping {
			trackID, isPlaying := GetCurrentTrackID()
			if !isPlaying {
				comm.ToChat(msg.Channel, "No track is currently playing.")
				return
			}

			// start bopping
			b.isBopping = true
			b.hasBopped[msg.User.Name] = true
			comm.ToChat(msg.Channel, fmt.Sprintf("BOP BOP BOP @%s has started the bopometer! Spam BOP to bop", msg.User.DisplayName))
			artists, _ := GetCurrentTrackArtists()
			song, _ := GetCurrentTrackTitle()
			b.currentTrack = trackInfo{Name: song, Artists: artists, Rating: 1, ID: trackID}
			comm.ToOverlay("bop start")
			c := make(chan int)
			go func(chan int) {
				comm.ToChat(msg.Channel, fmt.Sprintf("Bopping has %d seconds left! !bop away!", <-c))
			}(c)
			go func(chan int) {
				bopTimer(c)
				comm.ToChat(msg.Channel, "Bopping has concluded.")
				comm.ToOverlay("bop stop")
				b.isBopping = false
			}(c)
		} else {
			b.currentTrack.Rating++
		}
		return
	}
	if len(args) == 2 && args[1] == "top" {
		// get top bops
		ts := tracks{}
		for _, track := range b.ratings {
			ts = append(ts, track)
		}
		sort.Sort(ts)
		comm.ToChat(msg.Channel, "Top 3 BOPs:")
		for i := 0; i < 3; i++ {
			if i > len(ts) - 1 {
				comm.ToChat(msg.Channel, fmt.Sprintf("%d: ???", i+1))
				continue
			}
			artists := ""
			for k, a := range ts[i].Artists {
				artists += a
				if k < len(artists) - 1 {
					artists += ", "
				}
			}
			comm.ToChat(msg.Channel, fmt.Sprintf("%d: %s by %s with a %.2f rating.", i+1, ts[i].Name, artists, ts[i].Rating))
		}
	}
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
	for range ticker.C {
		i++
		if i == boppingWindow-bopEndWaringTime {
			c <- bopEndWaringTime
		}
		if i >= boppingWindow {
			return
		}
	}
}

func (b *Bopometer) GetBopping() bool {
	return b.isBopping
}

func (b *Bopometer) AddBops(n int) {
	//b.currentTrack.Rating += n
	comm.ToOverlay(fmt.Sprintf("bop add %d", n))
}

func (b *Bopometer) Results(args []string) {
	rating := args[2]
	res, err := strconv.ParseFloat(rating, 32)
	if err != nil {
		log.Println("invalid bop result from overlay", err)
	}
	b.currentTrack.Rating = float32(res)
	if track, exists := b.ratings[b.currentTrack.ID]; exists {
		if b.currentTrack.Rating > track.Rating {
			comm.ToChat("burtstanton", fmt.Sprintf("%s has set a new record with a %.2f rating on the Bopometer!", track.Name, b.currentTrack.Rating))
			b.ratings[b.currentTrack.ID] = b.currentTrack
		}
	} else {
		comm.ToChat("burtstanton", fmt.Sprintf("%s registered a rating of %.2f on the Bopometer!", b.currentTrack.Name, b.currentTrack.Rating))
		b.ratings[b.currentTrack.ID] = b.currentTrack
	}
	b.hasBopped = map[string]bool{}
	b.saveRatingsToFile()
}

func (b *Bopometer) Help() []string {
	return []string{
		"!bop will intiaite the bopometer",
		"spam BOP emotes to raise the bopometer",
		"Destroy stream quality in the process.",
	}
}

func (b *Bopometer) handleRawMessage(msg twitch.PrivateMessage) {
	if b.isBopping {
		bops := strings.Count(msg.Message, "BOP")
		b.AddBops(bops)
	}
}
