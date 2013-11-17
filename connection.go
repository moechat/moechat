package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
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
	User string `json:"u"`
	Message string `json:"m"`
}

func (c *connection) reader() {
	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			log.Println("Error receiving message: " + err.Error())
			break
		}
		log.Println("Message: " + string(message))
		smsg := strings.SplitN(string(message), ":", 2)
		code, msg := smsg[0], smsg[1]
		die := false
		switch code {
		default: log.Println("Code is not one of m, e, v and u. Code is: " + code)
		case "v": if(msg != CLIENT_VER) {
			c.ws.WriteMessage(websocket.TextMessage, []byte(`{"error":"Client out of date!"}`))
			log.Println("Client version out of date!")
			die = true
		}
		case "m":
			m := Message{User: c.CurrentUser.name, Message: string(msg)}
			msg, err := json.Marshal(m)
			if err != nil {
				log.Println("Error converting message to JSON: " + err.Error())
				break
			}
			h.broadcast <- []byte(msg)
		case "e": c.CurrentUser.Email = msg
		case "u": c.CurrentUser.Name = msg
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
			log.Println("Error sending message: " + err.Error())
			break
		}
		log.Println("Message sent")
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
	defer func() { h.unregister <- c }()
	go c.writer()
	c.reader()
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	ip := strings.Split(r.RemoteAddr,":")[0]
	log.Println("Handling request to /users from ip " + ip)

	users := []User{}

	for conn, _ := range h.connections {
		append(users, conn.CurrentUser)
	}

	json, err:= json.Marshal(users)
	if err != nil {
		log.Println("Failed to convert users to JSON: " + err.Error())
		return
	}

	fmt.Fprint(w, json)
}
