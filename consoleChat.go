package main 

import (
	"fmt"
	"log"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/gempir/go-twitch-irc/v2"
)

const numMessages = 7

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
			colorCode = 31
		case 60:
			colorCode = 33
		case 120:
			colorCode = 32
		case 180:
			colorCode = 36
		case 240:
			colorCode = 34
		case 300:
			colorCode = 35
		default:
			colorCode = 37
	}
	return fmt.Sprintf("\033[%dm", colorCode)
}

func showMessageOnConsole(msg twitch.PrivateMessage) {
	// don't show message that are commands
	if msg.Message[0] == '!' {
		return
	}
	chatMessages = append(chatMessages, msg)
	displayMessages()
}

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
		badges := ""
		msg := chatMessages[i]
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
		h, m, _ := chatMessages[i].Time.Clock()
		fmt.Println(fmt.Sprintf("%02d:%02d%s%s[%s]\033[0m: %s", 
			h, m, badges, cesc, msg.User.DisplayName, msg.Message))
	}
}

func deleteMessageByMsgID(id string) {
	// find it
	index := -1
	for i := 0; i < len(chatMessages); i++ {
		if chatMessages[i].ID == id {
			index = i
		}
	}
	if index == -1 {
		return
	}
	deleteChatMessage(index)
}

func handleClearChatMessage(message twitch.ClearChatMessage) {
	for i := 0; i < len(chatMessages); i++ {
		if chatMessages[i].User.ID == message.TargetUserID {
			deleteChatMessage(i)
			i--
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

func handleClearMessage(msg twitch.ClearMessage) {
	deleteMessageByMsgID(msg.TargetMsgID)
	displayMessages()
}

