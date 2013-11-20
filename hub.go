package main

import (
	"log"
)

type hub struct {
	connections map[*connection]int
	usernames map[string]bool

	broadcast chan []byte

	register chan *connection
	unregister chan *connection
}

var h = hub {
	broadcast: make(chan[] byte),
	register: make(chan *connection),
	unregister:  make(chan *connection),
	connections: make(map[*connection]int),
	usernames: make(map[string]bool),
}

var nextID = 1

func (h *hub) run() {
	for {
		select {
		case c := <-h.register:
			c.user.ID = nextID
			nextID += 1
			h.connections[c] = c.user.ID
			usersByID[c.user.ID] = c.user
			c.send(Command{"idset", map[string]string{"id":strconv.Itoa(c.user.ID)}})
			log.Printf("User with ip %s has joined.", c.ws.RemoteAddr())
		case c := <-h.unregister:
			if(c.user.Name != "") {
				log.Printf("User %s (ip %s) has left.", c.user.Name, c.ws.RemoteAddr())
			}
			delete(h.connections, c)
			delete(h.usernames, c.user.Name)
			close(c.toSend)
		case m := <-h.broadcast:
			for c := range h.connections {
				select {
				case c.toSend <- m:
				default:
					delete(h.connections, c)
					close(c.toSend)
					go c.ws.Close()
				}
			}
		}
	}
}
