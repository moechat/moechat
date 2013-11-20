package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"html"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type connection struct {
	// The websocket connection.
	ws *websocket.Conn
	// User info
	user User

	// Buffered channel of outbound messages.
	toSend chan []byte
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

func broadcast(v interface{}) {
	switch v := v.(type) {
	case Message: msg_logger.msgChan <- &v
	case Notification: msg_logger.notifChan <- &v
	}

	msg, err := json.Marshal(v);
	if err != nil {
		log.Printf("Error converting message %s to JSON: %v", msg, err)
		return
	}
	h.broadcast <- []byte(msg)
}

func (c *connection) send(v interface{}) {
	msg, err := json.Marshal(v);
	if err != nil {
		log.Printf("Error converting message %s to JSON: %v", msg, err)
		return
	}
	c.toSend <- []byte(msg)
}

func (c *connection) reader() {
	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			log.Println("Error receiving message: " + err.Error())
			break
		}
		log.Printf("Receiving message: %s", string(message))
		smsg := strings.SplitN(string(message), ":", 2)
		code, msg := smsg[0], smsg[1]
		die := false
		switch code {
		default: log.Println("Code is not one of m, e, v and u. Code is: " + code)
		case "v":
			if msg != CLIENT_VER {
				c.send(Error{"outofdate", `Client out of date! The most current version is <a href="moechat.sauyon.com">here</a>.`})
				log.Printf("Client version for ip %s out of date!", c.ws.RemoteAddr())
				die = true
			} else {
				broadcast(Notification{"User " + c.user.Name + " has joined the channel!"})
				broadcast(Command{"userjoin", map[string]string{"name":c.user.Name, "email":c.user.Email, "id":strconv.Itoa(c.user.ID)}})
			}
		case "m":
			broadcast(Message{User: c.user.Name, Message: msg})
		case "e":
			c.user.Email = msg
			broadcast(Command{"emailchange", map[string]string{"id":strconv.Itoa(c.user.ID), "email":msg}})
		case "u":
			if msg == "" || msg == c.user.Name {
				break
			}
			if len(msg) > 30 {
				msg = msg[:30]
				c.send(Notification{"Name is too long, your name will be set to "+msg})
				c.send(Command{"fnamechange", map[string]string{"newname":msg}})
			}
			delete(h.usernames, c.user.Name)
			used := h.usernames[msg]
			if used {
				num := 1
				nstr := strconv.Itoa(num)
				for h.usernames[msg+nstr] {
					num += 1
					nstr = strconv.Itoa(num)
				}
				c.send(Notification{"Name "+msg+" is taken, your name will be set to "+msg+nstr})
				c.send(Command{"fnamechange", map[string]string{"newname":msg+nstr}})
				msg = msg + nstr
			}
			msg = html.EscapeString(msg)
			if c.user.Name != "" {
				broadcast(Command{"namechange", map[string]string{"id":strconv.Itoa(c.user.ID), "newname":msg}})
				broadcast(Notification{"User " + c.user.Name + " is now known as " + msg})
			}
			c.user.Name = msg
			h.usernames[msg] = true;
		}
		if die {
			break
		}
	}
	c.ws.Close()
}

func (c *connection) writer() {
	for message := range c.toSend {
		err := c.ws.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("Error sending message to user %s: %v\n", c.user.Name, err)
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
	c := &connection{toSend: make(chan []byte, 256), user: User{"","",0}, ws: ws}
	h.register <- c
	defer func() {
		h.unregister <- c
		if c.user.Name == "" {
			return
		}
		broadcast(Command{"userleave", map[string]string{"id":strconv.Itoa(c.user.ID)}})
		broadcast(Notification{"User " + c.user.Name + " has left."})
	}()
	go c.writer()
	c.reader()
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	ip := strings.Split(r.RemoteAddr,":")[0]
	log.Println("Handling request to /users from ip " + ip)

	users := []User{}

	for conn, _ := range h.connections {
		users = append(users, conn.user)
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
