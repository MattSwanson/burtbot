package console 

import (
	"fmt"
	"os"
	"strconv"
	"log"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/gempir/go-twitch-irc/v2"
)

const numMessages = 25

const (
	Black = iota + 30
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
)

type nonChatMessage struct {
	Message		string
	ColorCode	int 
}

type ServiceStatus struct {
	Spotify bool
	Twitch bool
	Overlay bool
}

//var chatMessages []twitch.PrivateMessage
var chatMessages []interface{}
var status ServiceStatus

func SetServiceStatus(s ServiceStatus) {
	status = s
	displayMessages()
}

func SetSpotifyStatus(s bool) {
	status.Spotify = s
	displayMessages()
}


func SetTwitchStatus(s bool) {
	status.Twitch = s
	displayMessages()
}

func SetOverlayStatus(s bool) {
	status.Overlay = s
		displayMessages()
	}

func getColorEscapeCode(hexColor string) string {
	colorCode := 34
	if hexColor == "" {
		return fmt.Sprintf("\033[%dm", colorCode)
	}
	c, err := colorful.Hex(hexColor)
	if err != nil {
		log.Fatalf("couldn't convert %s to color", hexColor)
	}
	h, _, _ := c.Hsv()
	// clamp hues to specific values
	// r - 0, g - 120, b - 240, y - 60, c - 180, m - 300
	// white would be lum near 100, black near 0
	// round to the nearest 60deg value?
	ih := int(h)
	r := ih % 60
	if r < 30 {
		ih -= r
	} else {
		ih += 60 - r
	}
	ih = ih % 360
	switch ih {
		case 0:
			colorCode = Red
		case 60:
			colorCode = Yellow
		case 120:
			colorCode = Green
		case 180:
			colorCode = Cyan
		case 240:
			colorCode = Blue
		case 300:
			colorCode = Magenta
		default:
			colorCode = White
	}
	return fmt.Sprintf("\033[%dm", colorCode)
}

func ShowMessageOnConsole(msg twitch.PrivateMessage) {
	// don't show message that are commands
	if msg.Message[0] == '!' {
		return
	}
	chatMessages = append(chatMessages, msg)
	displayMessages()
}

// add ways to display errors/log lines in the chat window if wanted??

func displayMessages() {
	// print the last x messages to the screen - 5 for now
	fmt.Print("\033[H\033[2J")
	num := numMessages	// number of messages to display
	if num > len(chatMessages) {
		num = len(chatMessages)
	}
	start := len(chatMessages) - num
	if start < 0 {
		start = 0
	}
	end := start + num
	for i := start; i < end; i++ {
		switch msg := chatMessages[i].(type) {
			case twitch.PrivateMessage: 
				badges := ""
				if len(msg.User.Badges) > 0 {
					if _, ok := msg.User.Badges["broadcaster"]; ok {
						c := getColorEscapeCode("#FF0000")
						badges += fmt.Sprintf("%s[B]", c)
					}
					if _, ok := msg.User.Badges["moderator"]; ok {
						c := getColorEscapeCode("#00FF00")
						badges += fmt.Sprintf("%s[M]", c)
					}
				}
				cesc := getColorEscapeCode(msg.User.Color)
				h, m, _ := msg.Time.Clock()
				fmt.Println(fmt.Sprintf("%02d:%02d%s%s[%s]\033[0m: %s", 
					h, m, badges, cesc, msg.User.DisplayName, msg.Message))
			case nonChatMessage:
				fmt.Println(fmt.Sprintf("\033[%dm%s\033[0m", msg.ColorCode, msg.Message))
		}

	}

	// Draw the status bar at the bottom of the screen... maybe?
	spotifyStatusColor, twitchStatusColor, overlayStatusColor := Red, Red, Red
	if status.Spotify {
		spotifyStatusColor = Green
	}
	if status.Twitch {
		twitchStatusColor =  Green
	} 
	if status.Overlay {
		overlayStatusColor = Green
	}
	nColumns, _ := strconv.Atoi(os.Getenv("COLUMNS"))
	nSpaces := nColumns - 35
	// 35 columns for status text
	fmt.Printf("\033[100;0H\033[48;5;238m     \033[%dmSpotify    \033[%dmTwitch    \033[%dmOverlay",
		spotifyStatusColor, twitchStatusColor, overlayStatusColor)
	for i := 0; i < nSpaces; i++ {
		fmt.Print("-")
	}
	fmt.Printf("\033[0m\033[0G\033[1A")
}

func deleteMessageByMsgID(id string) {
	// find it
	index := -1
	for i := 0; i < len(chatMessages); i++ {
		switch msg := chatMessages[i].(type) {
			case twitch.PrivateMessage:
				if msg.ID == id {
					index = i
				}
		}
	}
	if index == -1 {
		return
	}
	deleteChatMessage(index)
}

func HandleClearChatMessage(message twitch.ClearChatMessage) {
	for i := 0; i < len(chatMessages); i++ {
		switch msg := chatMessages[i].(type) {
		case twitch.PrivateMessage:
			if msg.User.ID == message.TargetUserID {
				deleteChatMessage(i)
				i--
			}
		}
	}
	displayMessages()
}

func deleteChatMessage(index int) {
	if index == len(chatMessages)-1 {
		chatMessages = chatMessages[:index]
	} else {
		chatMessages = append(chatMessages[:index], chatMessages[index+1:]...)
	}	
}

func HandleClearMessage(msg twitch.ClearMessage) {
	deleteMessageByMsgID(msg.TargetMsgID)
	displayMessages()
}

// Display a message in the console chat
func AddMessage(msg string, colorCode int) {
	chatMessages = append(chatMessages, nonChatMessage{
		Message: msg,
		ColorCode: colorCode,
	})
	displayMessages()
}
