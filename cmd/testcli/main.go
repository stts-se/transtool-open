package main

import (
	"flag"
	"fmt"
	//"log"
	"os"
	//"regexp"
	"bytes"
	"strings"

	"github.com/gorilla/websocket"
)

//func jhjh() { fmt.Println() }

var socketBaseURL = "ws://localhost"

func connectSocket(port, clientID, user string) (*websocket.Conn, error) {
	var url = socketBaseURL + ":" + port + "/ws/" + clientID + "/" + user

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		//log.Fatal("Error connecting to Websocket Server:", err)
		return conn, fmt.Errorf("failed connecting to %s : %v", url, err)
	}
	//defer conn.Close()

	return conn, nil
}

func listen(conn *websocket.Conn) {
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read message from socket : %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("FROM SERVER:\t%s\n", string(msg))
	}

}

//var linesRE = regexp.MustCompile(`\n\n+`)

func main() {
	port := flag.String("port", "7372", "transtool server port")
	clientID := flag.String("client_id", "test_client", "client ID")
	user := flag.String("user", "anders_and", "user name")

	//jsonFile := flag.String("json_file", "", "file with JSON message")

	flag.Parse()

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "USAGE:\tgo run cmd/testcli/main.go cmd/testcli/list_audio_files.json <... json request files ...>\n")
		os.Exit(0)
	}

	//fmt.Println(*port)

	conn, err := connectSocket(*port, *clientID, *user)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connection failed : %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()
	go listen(conn)

	for _, fn := range os.Args[1:] {
		bts, err := os.ReadFile(fn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed reading file : %v\n", err)
			os.Exit(1)
		}

		for _, jsnObj := range bytes.Split(bts, []byte("\n\n")) {

			str := strings.TrimSpace(string(jsnObj))
			if str == "" {
				continue
			}

			//err = conn.WriteMessage(websocket.TextMessage, []byte(`{"message_type":"Ich bin ein Berliner"}`))
			err = conn.WriteMessage(websocket.TextMessage, jsnObj)
			//err = conn.WriteMessage(websocket.TextMessage, []byte(`},,,`))
			if err != nil {
				fmt.Fprintf(os.Stderr, "faile to write message to socket : %v\n", err)
				os.Exit(1)
			}
		}
	}

	select {}

}
