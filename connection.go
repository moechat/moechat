package main

import (
	"code.google.com/p/go.crypto/otr"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/moechat/moeparser"
	"log"
	"net/http"
	"strconv"
	"strings"
)

const (
	connected byte = iota
	versionChecked byte = iota
	joinedChannel byte = iota
	authenticated byte = iota
	closed byte = iota
)

func idToStr(id int64) string {
	return strconv.FormatInt(id, 10)
}

func strToId(str string) (int64, error) {
	return strconv.ParseInt(str, 10, 64)
}

type connection struct {
	// The websocket connection.
	ws *websocket.Conn
	// User info
	user *User
	target int64

	// Buffered channel of outbound messages.
	toSend chan []byte

	pongReceived bool

	state byte

	otr *otr.Conversation
}

var nextChatRoomId int64 = 1 << 62
type ChatRoom struct {
	Id int64
	Users map[*User]bool
	Type string
	Name string
}

var pmRooms map[*User]map[*User]*ChatRoom
var chatRooms []*ChatRoom = []*ChatRoom{lobby}

type Message struct {
	Sender int64 `json:"user"`
	Body string `json:"msg"`
	Target int64 `json:"target"`
}

type Notification struct {
	NotifBody string `json:"notif"`
	Targets []int64 `json:"targets,omitempty"`
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

	msgs := [][]byte{}
	if c.otr.IsEncrypted() {
		msgs, err = c.otr.Send(msg)
		if err != nil {
			log.Printf("Error encrypting messages %s: %v", msg, err)
			return
		}
	} else {
		msgs = [][]byte{msg}
	}

	for _,m := range msgs {
		select {
		case c.toSend <- m:
		default:
			c.state = closed
			delete(h.connections, c)
			delete(c.user.connections, c)
			close(c.toSend)
			go c.ws.Close()
		}
	}
}

func (c *connection) ping() {
	c.pongReceived = false
	c.toSend <- []byte{'p'}
}

func (c *connection) reader() {
ReadLoop:
	for c.state != closed {
		_, um, err := c.ws.ReadMessage()
		if err != nil {
			log.Println("Error receiving message:", err)
			break
		}

		message, _, change, toSend, err := c.otr.Receive(um)
		if err != nil {
			log.Println("Error unencrypting message:", err)
			continue
		}
		if change == otr.SMPComplete {
			log.Printf("SMP for user with ip %s succeeded.\n", c.ws.RemoteAddr())
		}
		if toSend != nil {
			log.Printf("toSend is not nil, sending otr message!")
			for _,msg := range toSend {
				select {
				case c.toSend <- msg:
				default:
					c.state = closed
					delete(h.connections, c)
					delete(c.user.connections, c)
					close(c.toSend)
					go c.ws.Close()
				}
			}
			continue
		}

		code, msg := byte(0), ""
		if len(message) > 1 {
			code, msg = message[0], string(message[1:])
		} else if len(message) == 1 {
			code, msg = message[0], ""
		} else {
			continue
		}

		if(code != 'p') {
			log.Printf("Receiving message %s:%s", string(code), string(msg))
		}
		die := false
		c.pongReceived = true
		switch code {
		default: log.Println("Code is not one of p, m, e, v and u. Code is: " + string(code))
		case 'p':
		case 'v':
			if msg != config.Version && msg != "0.13" {
				c.send(Error{"outofdate", `Client out of date! The most current version is <a href="//moechat.sauyon.com">here</a>.`})
				log.Printf("Client version for ip %s out of date!", c.ws.RemoteAddr())
				break ReadLoop
			} else {
				c.state = versionChecked
			}
		case 't':
			c.target, err = strToId(msg)
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
					oc.send(Message{c.user.Id, msg, c.user.Id})
				}
				for oc := range c.user.connections {
					oc.send(Message{c.user.Id, msg, c.target})
				}
			} else {
				broadcast(Message{c.user.Id, msg, 0})
			}
		case 'e':
			c.user.Email = msg
			if(c.state == joinedChannel) {
				broadcast(Command{"emailchange",
					map[string]string{
						"id":idToStr(c.user.Id),
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
				broadcast(Command{"namechange", map[string]string{"id":idToStr(c.user.Id), "newname":msg}})
				broadcast(Notification{
					"User " + c.user.Name + " is now known as " + msg,
					[]int64{0, c.user.Id}})
			}
			c.user.Name = msg
			h.usernames[msg] = true

			if c.state == versionChecked {
				if len(c.user.connections) == 1 {
					broadcast(Command{"userjoin",
						map[string]string{
							"name":c.user.Name,
							"email":c.user.Email, "id":idToStr(c.user.Id)}})
					broadcast(Notification{
						"User "+c.user.Name+" has joined the channel!",
						[]int64{0, c.user.Id}})
				}
				c.state = joinedChannel
			}
		//case 's':
		case 'k':
			if c.state >= joinedChannel {
				c.send(Command{"uploadkey", map[string]string{"key":genUploadKey(c.user.Id)}})
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
		otr: &otr.Conversation{PrivateKey: privKey},
	}
	c.user.connections[c] = true
	h.register <- c
	defer func() {
		if c.state == joinedChannel && len(c.user.connections) == 1 {
			h.unregister <- c
			broadcast(Notification{
				"User " + c.user.Name + " has left.",
				[]int64{0, c.user.Id}})
			broadcast(Command{"userleave", map[string]string{"id":idToStr(c.user.Id)}})
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
