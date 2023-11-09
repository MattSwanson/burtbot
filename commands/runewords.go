package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

/*type rune struct {
	number string
	name string
}*/

var runes = map[string]string{
	"r01": "El",
	"r02": "Eld",
	"r03": "Tir",
	"r04": "Nef",
	"r05": "Eth",
	"r06": "Ith",
	"r07": "Tal",
	"r08": "Ral",
	"r09": "Ort",
	"r10": "Thul",
	"r11": "Amn",
	"r12": "Sol",
	"r13": "Shael",
	"r14": "Dol",
	"r15": "Hel",
	"r16": "Io",
	"r17": "Lum",
	"r18": "Ko",
	"r19": "Fal",
	"r20": "Lem",
	"r21": "Pul",
	"r22": "Um",
	"r23": "Mal",
	"r24": "Ist",
	"r25": "Gul",
	"r26": "Vex",
	"r27": "Ohm",
	"r28": "Lo",
	"r29": "Sur",
	"r30": "Ber",
	"r31": "Jah",
	"r32": "Cham",
	"r33": "Zod",
}

/*var runes []rune{
	rune{"r01", "El"},
	rune{"r02", "Eld"},
	rune{"r03", "Tir"},
	rune{"r04", "Nef"},
	rune{"r05", "Eth"},
	rune{"r06", "Ith"},
	rune{"r07", "Tal"},
	rune{"r08", "Ral"},
	rune{"r09", "Ort"},
	rune{"r10", "Thul"},
	rune{"r11", "Amn"},
	rune{"r12", "Sol"},
	rune{"r13", "Shael"},
	rune{"r14", "Dol"},
	rune{"r15", "Hel"},
	rune{"r16", "Io"},
	rune{"r17", "Lum"},
	rune{"r18", "Ko"},
	rune{"r19", "Fal"},
	rune{"r20", "Lem"},
	rune{"r21", "Pul"},
	rune{"r22", "Um"},
	rune{"r23", "Mal"},
	rune{"r24", "Ist"},
	rune{"r25", "Gul"},
	rune{"r26", "Vex"},
	rune{"r27", "Ohm"},
	rune{"r28", "Lo"},
	rune{"r29", "Sur"}
	rune{"r30", "Ber"}
	rune{"r31", "Jah"}
	rune{"r32", "Cham"}
	rune{"r33", "Zod"}
}*/

type runew struct {
	Name   string `json:"Rune Name"`
	Runes  string `json:"*runes"`
	Rune1  string
	Rune2  string
	Rune3  string
	Rune4  string
	Rune5  string
	Rune6  string
	Server string
}

type Runeword struct{}

var runeWord = &Runeword{}

func init() {
	RegisterCommand("rw", runeWord)
}

func (rw *Runeword) PostInit() {

}

func (rw *Runeword) Run(msg twitch.PrivateMessage) {
	args := strings.Fields(strings.ToLower(msg.Message))
	if len(args) < 2 {
		return
	}
	search := strings.Join(args[1:], " ")
	req, err := http.NewRequest("GET", "https://d2api.netlify.app/api/runes.json", nil)
	if err != nil {
		requestErr(msg.Channel)
		log.Println("Couldn't create request: ", err.Error())
		return
	}
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		requestErr(msg.Channel)
		log.Println("Error making http request: ", err.Error())
		return
	}
	defer resp.Body.Close()
	rws := []runew{}
	err = json.NewDecoder(resp.Body).Decode(&rws)
	if err != nil {
		requestErr(msg.Channel)
		log.Println("Couldn't decode JSON from response body: ", err.Error())
		return
	}

	for _, r := range rws {
		if strings.ToLower(r.Name) == search {
			comm.ToChat(msg.Channel, r.Runes)
			runesInWord := ""
			if r.Rune1 != "" {
				runesInWord += runes[r.Rune1]
			}
			if r.Rune2 != "" {
				runesInWord += runes[r.Rune2]
			}
			if r.Rune3 != "" {
				runesInWord += runes[r.Rune3]
			}
			if r.Rune4 != "" {
				runesInWord += runes[r.Rune4]
			}
			if r.Rune5 != "" {
				runesInWord += runes[r.Rune5]
			}
			if r.Rune6 != "" {
				runesInWord += runes[r.Rune6]
			}
			if r.Server != "" {
				comm.ToChat(msg.Channel, "Ladder Only")
			}
			return
		}
	}
	comm.ToChat(msg.Channel, fmt.Sprintf("Couldn't find runeword %s.", args[1]))
}

func requestErr(channel string) {
	comm.ToChat(channel, "There was an error making the request. Try again later.")
}

func (rw *Runeword) Help() []string {
	return []string{
		"!rw <runeword name> to search for a d2 runeword",
	}
}
