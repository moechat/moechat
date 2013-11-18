package main

import (
	"log"
)

type hub struct {
	connections map[*connection]bool

	broadcast chan []byte

	register chan *connection
	unregister chan *connection
}

var h = hub {
	broadcast: make(chan[] byte),
	register: make(chan *connection),
	unregister:  make(chan *connection),
	connections: make(map[*connection]bool),
}

func (h *hub) run() {
	for {
		select {
		case c := <-h.register:
			h.connections[c] = true
			log.Printf("User with ip %s has joined.", c.ws.RemoteAddr())
		case c := <-h.unregister:
			log.Printf("User %s (ip %s) has left.", c.CurrentUser.Name, c.ws.RemoteAddr())
			delete(h.connections, c)
			close(c.send)
		case m := <-h.broadcast:
			for c := range h.connections {
				select {
				case c.send <- m:
				default:
					delete(h.connections, c)
					close(c.send)
					go c.ws.Close()
				}
			}
		}
	}
}
