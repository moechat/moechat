package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"html"
	"log"
	"net/http"
	"strings"
)

type connection struct {
	// The websocket connection.
	ws *websocket.Conn
	// User info
	CurrentUser User

	// Buffered channel of outbound messages.
	send chan []byte
}

type Message struct {
	User string `json:"user"`
	Message string `json:"msg"`
}

type Notification struct {
	NotifBody string `json:"notif"`
}

type Error struct {
	ErrorType string `json:"error"`
	ErrorMsg string `json:"msg"`
}

type Command struct {
	Command string `json:"cmd"`
	Args map[string]string `json:"args"`
}

func Broadcast(v interface{}) {
	msg, err := json.Marshal(v);
	if err != nil {
		log.Printf("Error converting message %s to JSON: %v", msg, err)
		return
	}
	h.broadcast <- []byte(msg)
}

func (c *connection) Send(v interface{}) {
	msg, err := json.Marshal(v);
	if err != nil {
		log.Printf("Error converting message %s to JSON: %v", msg, err)
		return
	}
	c.send <- []byte(msg)
}

func (c *connection) reader() {
	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			log.Println("Error receiving message: " + err.Error())
			break
		}
		smsg := strings.SplitN(string(message), ":", 2)
		code, msg := smsg[0], smsg[1]
		die := false
		switch code {
		default: log.Println("Code is not one of m, e, v and u. Code is: " + code)
		case "v":
			if(msg != CLIENT_VER) {
				c.Send(Error{"outofdate", "Client out of date!"})
				log.Printf("Client version for ip %s out of date!", c.ws.RemoteAddr())
				die = true
			} else {
				Broadcast(Notification{"User " + c.CurrentUser.Name + " has joined the channel!"})
				Broadcast(Command{"userjoin", map[string]string{"name":c.CurentUser.Name, "email":c.CurrentUser.Email}})
			}
		case "m":
			Broadcast(Message{User: c.CurrentUser.Name, Message: msg})
		case "e": c.CurrentUser.Email = msg
		case "u":
			if(msg != "" && msg != c.CurrentUser.Name) {
				msg = html.EscapeString(msg)
				if(c.CurrentUser.Name != "") {
					Broadcast(Command{"namechange", map[string]string{"currname":c.CurrentUser.Name, "email":c.CurrentUser.Email, "newname":msg}})
					Broadcast(Notification{"User " + c.CurrentUser.Name + " is now known as " + msg})
				}
				c.CurrentUser.Name = msg
			}
		}
		if(die) {
			break
		}
	}
	c.ws.Close()
}

func (c *connection) writer() {
	for message := range c.send {
		err := c.ws.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("Error sending message to user %s: %v\n", c.CurrentUser.Name, err)
			break
		}
	}
	c.ws.Close()
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	ip := strings.Split(r.RemoteAddr,":")[0]
	log.Println("Handling request to /chat from ip " + ip)
	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		log.Println("Non-websocket request sent to /chat, dying")
		return
	} else if err != nil {
		log.Println("Error handling /chat request: " + err.Error())
		return
	}
	c := &connection{send: make(chan []byte, 256), CurrentUser: User{"",""}, ws: ws}
	h.register <- c
	defer func() {
		h.unregister <- c
		Broadcast(Command{"userleave", map[string]string{"name":c.CurrentUser.Name, "email":c.CurrentUser.Email}})
		Broadcast(Notification{"User " + c.CurrentUser.Name + " has left."})
	}()
	go c.writer()
	c.reader()
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	ip := strings.Split(r.RemoteAddr,":")[0]
	log.Println("Handling request to /users from ip " + ip)

	users := []User{}

	for conn, _ := range h.connections {
		users = append(users, conn.CurrentUser)
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	err := enc.Encode(users)
	if err != nil {
		log.Println("Failed to convert users to JSON: " + err.Error())
		return
	}
}
