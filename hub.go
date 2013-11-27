package main

import (
	"log"
	"time"
)

type hub struct {
	connections map[*connection]int64
	usernames map[string]bool

	broadcast chan []byte

	register chan *connection
	unregister chan *connection

	timeoutTicker *time.Ticker
}

var h = hub {
	broadcast: make(chan[] byte),
	register: make(chan *connection),
	unregister:  make(chan *connection),
	connections: make(map[*connection]int64),
	usernames: make(map[string]bool),

	timeoutTicker: time.NewTicker(10 * time.Second),
}

var lobby = &ChatRoom{0, make(map[*User]bool), "", "lobby"}

var nextAnonId int64 = -1
func (h *hub) run() {
	for {
		select {
		case c := <-h.register:
			c.user.Id = nextAnonId
			nextAnonId--
			h.connections[c] = c.user.Id
			usersById[c.user.Id] = c.user
			c.send(Command{"idset", map[string]string{"id":idToStr(c.user.Id)}})
			log.Printf("User with ip %s has joined.", c.ws.RemoteAddr())
		case c := <-h.unregister:
			if(c.state == joinedChannel) {
				log.Printf("User %s (ip %s) has left.", c.user.Name, c.ws.RemoteAddr())
			} else if(c.state == closed) {
				break
			}
			c.state = closed
			delete(h.connections, c)
			delete(c.user.connections, c)
			delete(h.usernames, c.user.Name)
			close(c.toSend)
		case <-h.timeoutTicker.C:
			for c := range h.connections {
				if(c.pongReceived) {
					c.ping()
				} else {
					if(c.state == joinedChannel) {
						log.Printf("User %s (ip %s) timed out", c.user.Name, c.ws.RemoteAddr())
					}
					if c.state == joinedChannel && len(c.user.connections) == 1 {
						go broadcast(Notification{
							"User " + c.user.Name + " has timed out.",
							[]int64{0, c.user.Id}})
						go broadcast(Command{"userleave", map[string]string{"id":idToStr(c.user.Id)}})
					}
					c.state = closed
					delete(h.connections, c)
					delete(c.user.connections, c)
					delete(h.usernames, c.user.Name)
					close(c.toSend)
				}
			}
		case m := <-h.broadcast:
			for c := range h.connections {
				select {
				case c.toSend <- m:
				default:
					delete(h.connections, c)
					delete(c.user.connections, c)
					close(c.toSend)
					go c.ws.Close()
				}
			}
		}
	}
}
