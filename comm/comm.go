package comm

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/MattSwanson/burtbot/console"
	"github.com/gempir/go-twitch-irc/v2"
)

var writeChannel chan string
var readChannel chan string
var inputChannel chan string
var subscribers map[string][]func([]string)
var chatClient *twitch.Client
var connectedToOverlay bool

func init() {
	writeChannel = make(chan string)
	readChannel = make(chan string)
	inputChannel = make(chan string)
	subscribers = make(map[string][]func([]string))
	//listenOnInput()
}

func GetReadChannel() chan string {
	return readChannel
}

func GetInputChannel() chan string {
	return inputChannel
}

func ToOverlay(s string) {
	// keep things from queuing up when connected
	// and also from goroutines from hanging around
	// waiting for the channel to read
	if IsConnectedToOverlay() {
		writeChannel <- s
	}
}

func FromOverlay() string {
	return <-readChannel
}

func AddChatClient(client *twitch.Client) {
	chatClient = client
}

func ToChat(channelName string, msg string) {
	chatClient.Say(channelName, msg)
}

func listenOnInput() {
	go func(c chan string) {
		scanner := bufio.NewScanner(os.Stdin)
		for {
			scanner.Scan()
			text := scanner.Text()
			if text == "" {
				continue
			}
			ToChat("burtstanton", text)
		}
	}(inputChannel)
}

func ConnectToOverlay() {
	addr := fmt.Sprintf("%s:8081", os.Getenv("OVERLAY_IP"))
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		//log.Println("Couldn't connect to overlay")
		time.Sleep(time.Second * 10)
		ConnectToOverlay()
		return
	}
	defer conn.Close()
	go func() {
		getMessagesFromTCP(conn)
	}()
	go notifySubscribers()
	ctx, cancelPing := context.WithCancel(context.Background())
	pingOverlay(ctx, writeChannel)
	console.SetOverlayStatus(true)
	connectedToOverlay = true
	for {
		s := <-writeChannel
		// fmt.Println(s)
		_, err := fmt.Fprintf(conn, "%s\n", s)
		if err != nil {
			// we know we have no connection, stop pinging until we reconnect
			cancelPing()
			connectedToOverlay = false
			console.SetOverlayStatus(false)
			log.Println("Lost connection to overlay... will retry in 5 sec.")
			readChannel <- "reset"
			time.Sleep(time.Second * 5)
			ConnectToOverlay()
		}
	}
}

// put a ping on the comm channel every 3 seconds to make sure we still have a connection
func pingOverlay(ctx context.Context, c chan string) {
	go func(ctx context.Context) {
		t := time.NewTicker(time.Second * 3)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				c <- "ping"
			}
		}
	}(ctx)
}

func SubscribeToReply(command string, f func([]string)) {
	if _, ok := subscribers[command]; !ok {
		subscribers[command] = []func([]string){}
	}
	subscribers[command] = append(subscribers[command], f)
}

// operating on its own goroutine
func getMessagesFromTCP(conn net.Conn) {
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		s := scanner.Text()
		readChannel <- s
	}
	if err := scanner.Err(); err != nil {
		log.Println(err)
	}
}

func notifySubscribers() {
	for s := range readChannel {
		args := strings.Fields(s)
		for _, f := range subscribers[args[0]] {
			f(args)
		}
		if args[0] == "reset" {
			break
		}
	}
}

// IsConnetedToOverlay allows users of ToOverlay to check if
// a connection is active. ToOverlay will automatically drop
// strings sent if there is no connection to prevent queueing(?)
// and to keep goroutines from hanging waiting for the channel to
// be read from.
//
// Different behaviour may be wanted if there overlay is not connected
// Hence exporting this function. Also, you may want to know the connection
// status before calling ToChat so that's why ToChat doesn't return an
// error or bool or something like that
func IsConnectedToOverlay() bool {
	return connectedToOverlay
}
