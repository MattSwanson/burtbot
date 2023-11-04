package commands

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/gempir/go-twitch-irc/v2"
)

type metar struct{}

var mc *metar = &metar{}
var awcEndpoint = "https://www.aviationweather.gov/cgi-bin/data/metar.php"

type awcApiResponse struct {
	RawMetar string `xml:"data>METAR>raw_text"`
}

func init() {
	RegisterCommand("metar", mc)
}

func (m *metar) PostInit() {}

func (m *metar) Run(msg twitch.PrivateMessage) {
	args := strings.Fields(strings.ToLower(strings.TrimPrefix(msg.Message, "!")))
	if len(args) < 2 {
		comm.ToChat(msg.Channel, "Insufficient Arguments. Learn to use it please. No help from me...")
		return
	}

	metar := getMetar(args[1])
	if metar == "" {
		comm.ToChat(msg.Channel, "Invalid airport identifier")
	}
	comm.ToChat(msg.Channel, metar)
}

func getMetar(stationID string) string {
	url := fmt.Sprintf("%s?ids=%s", awcEndpoint, stationID)
	resp, err := http.Get(url)

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "that did not work"
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return string(bodyBytes)
}

func (m *metar) Help() []string {
	return []string{"Get the current METAR for an airport"}
}
