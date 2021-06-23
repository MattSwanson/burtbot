package comm

import (
	"fmt"
	"net"
	"log"
	"time"
	"context"
	"bufio"
	"os"
)

var writeChannel chan string
var readChannel chan string

func init() {
	writeChannel = make(chan string)
	readChannel = make(chan string)
}

func GetReadChannel() chan string {
	return readChannel
}

func ToOverlay(s string) {
	writeChannel <- s
}

func FromOverlay() string {
	return <-readChannel
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
	ctx, cancelPing := context.WithCancel(context.Background())
	pingOverlay(ctx, writeChannel)
	fmt.Println("Connected to overlay")
	for {
		s := <-writeChannel
		// fmt.Println(s)
		_, err := fmt.Fprintf(conn, "%s\n", s)
		if err != nil {
			// we know we have no connection, stop pinging until we reconnect
			cancelPing()
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

func getMessagesFromTCP(conn net.Conn) {
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		s := scanner.Text()
		//fmt.Println(s)
		readChannel <- s
	}
	if err := scanner.Err(); err != nil {
		log.Println(err)
	}
}
