package commands

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/MattSwanson/burtbot/db"
	"github.com/gempir/go-twitch-irc/v2"
)

// here lies the bopometer...
type Bopometer struct {
	currentTrack db.BopRating
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

var bopometer *Bopometer = &Bopometer{}

func init() {
	bopometer.hasBopped = map[string]bool{}
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
			artistsSlice, _ := GetCurrentTrackArtists()
			artists := ""
			for i := 0; i < len(artistsSlice); i++ {
				artists += artistsSlice[i]
				if i < len(artistsSlice) -1 {
					artists += ", "
				}
			}
			song, _ := GetCurrentTrackTitle()
			b.currentTrack = db.BopRating{
				SongName: song, 
				SongArtists: artists,  
				SpotifyID: trackID,
				AddedAt: time.Now(),
			}
			comm.ToOverlay("bop start")
			c := make(chan int)
			go func(chan int) {
				comm.ToChat(msg.Channel, fmt.Sprintf("Bopping has %d seconds left! BOP away!", <-c))
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
		ts, err := db.GetBopRatings(3)
		if err != nil {
			comm.ToChat(msg.Channel, "Can't get top bops at the moment. Try again some other time when it works.")
			log.Println("couldn't get 3 bops: ", err)
			return
		}
		comm.ToChat(msg.Channel, "Top 3 BOPs:")
		for i := 0; i < 3; i++ {
			if i > len(ts) - 1 {
				comm.ToChat(msg.Channel, fmt.Sprintf("%d: ???", i+1))
				continue
			}
			comm.ToChat(msg.Channel, fmt.Sprintf("%d: %s by %s with a %.2f rating.", i+1, ts[i].SongName, ts[i].SongArtists, ts[i].Rating))
		}
	}
}

// search rating by track / artist
// top 3 or 5
// 	- track title/artists and rating

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
	comm.ToOverlay(fmt.Sprintf("bop add %d", n))
}

func (b *Bopometer) Results(args []string) {
	rating := args[2]
	res, err := strconv.ParseFloat(rating, 32)
	if err != nil {
		log.Println("invalid bop result from overlay", err)
	}
	b.currentTrack.Rating = float32(res)
	if track, _ := db.GetBopRating(b.currentTrack.SpotifyID); track.SpotifyID != "" {
		if b.currentTrack.Rating > track.Rating {
			comm.ToChat("burtstanton", fmt.Sprintf("%s has set a new record with a %.2f rating on the Bopometer!", track.SongName, b.currentTrack.Rating))
			if err := db.UpdateBopRating(b.currentTrack.SpotifyID, b.currentTrack.Rating); err != nil {
				log.Println("couldn't update bop rating in db: ", err)
			}
		}
	} else {
		comm.ToChat("burtstanton", fmt.Sprintf("%s registered a rating of %.2f on the Bopometer!", b.currentTrack.SongName, b.currentTrack.Rating))
		if err := db.AddBopRating(b.currentTrack); err != nil {
			log.Println("couldn't add bop to db: ", err)
		}
	}
	b.hasBopped = map[string]bool{}
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
