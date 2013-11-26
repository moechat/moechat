package main

import (
	//"code.google.com/p/go.crypto/otr"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/moechat/moeparser"
	"log"
	"net/http"
	"strconv"
	"strings"
)

const (
	connected byte = 0
	versionChecked byte = 1
	joinedChannel byte = 2
	closed byte = 3
)

type connection struct {
	// The websocket connection.
	ws *websocket.Conn
	// User info
	user *User
	target int

	// Buffered channel of outbound messages.
	toSend chan []byte

	pongReceived bool

	state byte
}

var nextChatRoomId = 1
type ChatRoom struct {
	Id int
	Users map[*User]bool
	Type string
	Name string
}

var pmRooms map[*User]map[*User]*ChatRoom
var chatRooms []*ChatRoom = []*ChatRoom{lobby}

type Message struct {
	Sender int `json:"user"`
	Body string `json:"msg"`
	Targets []int `json:"targets,omitempty"`
}

type Notification struct {
	NotifBody string `json:"notif"`
	Targets []int `json:"targets,omitempty"`
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
	select {
	case c.toSend <- []byte(msg):
	default:
		delete(h.connections, c)
		delete(c.user.connections, c)
		close(c.toSend)
		go c.ws.Close()
	}
}

func (c *connection) ping() {
	c.pongReceived = false
	c.toSend <- []byte{'p'}
}

func (c *connection) reader() {
ReadLoop:
	for c.state != closed {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			log.Println("Error receiving message: " + err.Error())
			break
		}
		code, msg := message[0], string(message[1:])
		if(code != 'p') {
			log.Printf("Receiving message %s: %s", string(code), string(msg))
		}
		die := false
		switch code {
		default: log.Println("Code is not one of p, m, e, v and u. Code is: " + string(code))
		case 'p': c.pongReceived = true
		case 'v':
			if msg != ClientVer {
				c.send(Error{"outofdate", `Client out of date! The most current version is <a href="//moechat.sauyon.com">here</a>.`})
				log.Printf("Client version for ip %s out of date!", c.ws.RemoteAddr())
				break ReadLoop
			} else {
				c.state = versionChecked
			}
		case 't':
			c.target, err = strconv.Atoi(msg)
			if err != nil {
				log.Println("Error setting target:", err)
			}
		case 'm':
			if c.state != joinedChannel {
				break
			}

			msg, err = moeparser.Parse(msg)
			if err != nil {
				log.Println("Error parsing message:", err)
			}

			if(c.target != 0) {
				for oc := range getUser(c.target).connections {
					oc.send(Message{c.user.ID, msg, []int{c.user.ID}})
				}
				for oc := range c.user.connections {
					oc.send(Message{c.user.ID, msg, []int{c.target}})
				}
			} else {
				broadcast(Message{c.user.ID, msg, []int{0}})
			}
		case 'e':
			c.user.Email = msg
			if(c.state == joinedChannel) {
				broadcast(Command{"emailchange",
					map[string]string{
						"id":strconv.Itoa(c.user.ID),
						"email":msg}})
			}
		case 'u':
			msg = strings.TrimSpace(msg)
			if c.state < versionChecked {
				log.Printf("User %s (ip %s) attempted to set a name before version checking", c.user.Name, c.ws.RemoteAddr())
				break ReadLoop
			} else if msg == "" || msg == c.user.Name {
				break
			}

			if len(msg) > 30 {
				msg = msg[:30]
				c.send(Notification{NotifBody: "Name is too long, your name will be set to "+msg})
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
				c.send(Notification{NotifBody: "Name "+msg+" is taken, your name will be set to "+msg+nstr})
				c.send(Command{"fnamechange", map[string]string{"newname":msg+nstr}})
				msg = msg + nstr
			}
			if c.user.Name != "" {
				broadcast(Command{"namechange", map[string]string{"id":strconv.Itoa(c.user.ID), "newname":msg}})
				broadcast(Notification{
					"User " + c.user.Name + " is now known as " + msg,
					[]int{0, c.user.ID}})
			}
			c.user.Name = msg
			h.usernames[msg] = true

			if c.state == versionChecked {
				if len(c.user.connections) == 1 {
					broadcast(Command{"userjoin",
						map[string]string{
							"name":c.user.Name,
							"email":c.user.Email, "id":strconv.Itoa(c.user.ID)}})
					broadcast(Notification{
						"User "+c.user.Name+" has joined the channel!",
						[]int{0, c.user.ID}})
				}
				c.state = joinedChannel
			}
		}

		if die {
			break
		}
	}
	c.ws.Close()
}

func (c *connection) writer() {
	for message := range c.toSend {
		if c.state == closed {
			break
		}

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
	c := &connection{
		toSend: make(chan []byte, 256),
		user: &User{connections: make(map[*connection]bool)},
		ws: ws,
		pongReceived: true,
	}
	c.user.connections[c] = true
	h.register <- c
	defer func() {
		if c.state == joinedChannel && len(c.user.connections) == 1 {
			h.unregister <- c
			broadcast(Notification{
				"User " + c.user.Name + " has left.",
				[]int{0, c.user.ID}})
			broadcast(Command{"userleave", map[string]string{"id":strconv.Itoa(c.user.ID)}})
		} else {
			h.unregister <- c
		}
	}()
	go c.writer()
	c.reader()
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	ip := strings.Split(r.RemoteAddr,":")[0]
	log.Println("Handling request to /users from ip " + ip)

	users := []*User{}

	for conn, _ := range h.connections {
		users = append(users, conn.user)
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "X-Requested-With")
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	err := enc.Encode(users)
	if err != nil {
		log.Println("Failed to convert users to JSON: " + err.Error())
		return
	}
}
