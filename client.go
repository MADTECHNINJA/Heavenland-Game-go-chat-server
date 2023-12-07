package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 2048

	// actions
	actionLogin   = "login"
	actionMessage = "message"
)

var (
	newline = []byte{'\n'}
	// space          = []byte{' '}
	authError      = []byte(`{"error": "you need to authenticate first"}`)
	actionError    = []byte(`{"error": "valid action need to be choosen"}`)
	tokenInvalid   = []byte(`{"error": "access token is invalid"}`)
	tokenValid     = []byte(`{"tokenValid": true, "info": "fetching username"}`)
	loginUnsuccess = []byte(`{"error": "failed fetching username, login unsuccessful"}`)
	loginSuccess   = []byte(`{"info": "connected"}`)
)

// define upgrader to limit maximum size of read and write buffer
var upgrader = websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
}

type Message struct {
	Action    string `json:"action"`
	Token     string `json:"token,omitempty"`
	Message   string `json:"message"`
	Channel   string `json:"channel"`
	Timestamp int64  `json:"timestamp"`
	Nickname  string `json:"nickname"`
}

type ServerInfo struct {
	Action  string `json:"action"`
	Message string `json:"message"`
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	server *Server

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// boolean status of authentication
	auth bool

	// string with the userId
	userId string

	// nickname
	nickname string
}

// readPump pumps messages from the websocket connection to the server
func (c *Client) readPump() {
	defer func() {
		c.server.unregister <- c
		// c.server.broadcast <- broadcast_server_message(c.nickname + " disconnected from the server")
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		// parse the incoming message
		message := Message{}
		json.Unmarshal([]byte(data), &message)

		// check for login message
		if !c.auth && message.Action == actionLogin {
			heavenland := newHeavneland()
			statement := heavenland.authenticate(message.Token)
			if statement {
				c.send <- tokenValid
				c.auth = heavenland.auth
				c.userId = heavenland.userId
			} else {
				c.send <- tokenInvalid
			}
			if c.auth {
				statement, data := heavenland.fetchUsername(message.Token)
				if statement {
					c.nickname = data.Nickname
					c.send <- loginSuccess
					// c.server.broadcast <- broadcast_server_message(c.nickname + " connected to the server")
				} else {
					c.send <- loginUnsuccess
				}
			}
			continue
		} else if !c.auth {
			c.send <- authError
			continue
		}

		// check for other messages
		switch message.Action {
		case actionMessage:
			message.Nickname = c.nickname
			message.Timestamp = time.Now().Unix()
			data, err = json.Marshal(message)
			if err != nil {
				fmt.Println("error on parsing json to bytes")
				continue
			}
			c.server.broadcast <- data
		default:
			c.send <- actionError
		}
	}
}

// writePump pumps messages from the server to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func serveWs(server *Server, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// create a new client
	client := &Client{server: server, conn: conn, send: make(chan []byte, 256), auth: false}
	client.server.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}

// func broadcast_server_message(message string) []byte {
// 	serverInfo := ServerInfo{}
// 	serverInfo.Action = "serverInfo"
// 	serverInfo.Message = message

// 	data, err := json.Marshal(serverInfo)
// 	if err != nil {
// 		fmt.Println("error parsing json at broadcast connect")
// 	}
// 	return data
// }
