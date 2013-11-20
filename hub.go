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

var nextID = 0

func (h *hub) run() {
	for {
		select {
		case c := <-h.register:
			h.connections[c] = nextID
			nextID += 1
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
