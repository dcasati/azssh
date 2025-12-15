package azssh

import (
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/gorilla/websocket"
	"golang.org/x/term"
)

func dial(url string, token string) *websocket.Conn {
	headers := http.Header{}
	// Service Bus Relay websockets don't use Bearer tokens
	// The authentication is handled through the relay URL itself
	
	dialer := websocket.Dialer{}
	
	c, resp, err := dialer.Dial(url, headers)
	if err != nil {
		if resp != nil {
			log.Printf("dial failed with status %d\n", resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			if len(body) > 0 {
				log.Printf("response body: %s\n", string(body))
			}
		}
		log.Fatal("dial:", err)
	}
	return c
}

// read from ws
func pumpOutput(c *websocket.Conn, w io.Writer, done chan struct{}) {
	defer close(done)
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		w.Write(message)
	}
}

// send to ws
func pumpInput(c *websocket.Conn, r io.Reader) {
	data := make([]byte, 1)
	for {
		r.Read(data)

		err := c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Println("send:", err)
		}
	}
}

func pumpInterrupt(c *websocket.Conn) {
	var interrupt = make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	for {
		<-interrupt
		sigint := []byte{'\003'}
		c.WriteMessage(websocket.TextMessage, sigint)
	}
}

// GetTerminalSize gets the size of the current terminal
func GetTerminalSize() TerminalSize {
	cols, rows, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		cols = 80
		rows = 30
	}
	return TerminalSize{
		Rows: rows,
		Cols: cols,
	}
}

// ConnectToWebsocket wires up STDIN and STDOUT to a websocket, allowing you to use it as a terminal
func ConnectToWebsocket(url string, token string, resize chan<- TerminalSize) {
	// Save the current terminal state
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatal("Failed to set raw mode:", err)
	}
	
	// Ensure terminal state is restored on exit
	defer func() {
		term.Restore(int(os.Stdin.Fd()), oldState)
		// Move cursor to a new line on exit for clean prompt
		os.Stdout.Write([]byte("\r\n"))
	}()

	// hook into terminal resizes
	go pumpResize(resize)

	done := make(chan struct{})

	c := dial(url, token)
	defer c.Close()

	go pumpOutput(c, os.Stdout, done)
	go pumpInput(c, os.Stdin)
	go pumpInterrupt(c)

	<-done
}
