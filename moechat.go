package main

import (
	"net/http"
	"log"
	"os"
	"strings"
)

var CLIENT_VER = "0.8"
var LOG_FILE = "/var/log/moechat.log"
var MSG_LOG_FILE = "/root/messages.log"

func handler(w http.ResponseWriter, r *http.Request) {
	ip := strings.Split(r.RemoteAddr,":")[0]
	if ip == "54.227.38.194" {
		return
	}
	log.Println("Handling connection from ip " + ip + " for: " + r.URL.Path)
	name := "/srv/chat/index.html"
	if r.URL.Path != "/" {
		name = "/srv/chat" + r.URL.Path
	}
	file, err := os.Open(name)
	if err != nil {
		errorHandler(w, r, err)
		return
	}
	finfo, err := file.Stat()
	if err != nil {
		errorHandler(w, r, err)
		return
	}
	http.ServeContent(w, r, name, finfo.ModTime(), file)
}

func errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	log.Println("Error handling request: " + err.Error())
	if os.IsNotExist(err) {
		name := "/srv/chat/404.html"
		file, err := os.Open(name)
		if err != nil {
			log.Println("Error opening 404: " + err.Error())
			return
		}
		finfo, err := file.Stat()
		if err != nil {
			log.Println("Error opening 404: " + err.Error())
			return
		}
		http.ServeContent(w, r, name, finfo.ModTime(), file)
	}
}

type MessageLog struct {
	MsgChan chan *Message
	NotifChan chan *Notification
}

var msg_logger = MessageLog {
	MsgChan: make(chan *Message),
	NotifChan: make(chan *Notification),
}
func (logger *MessageLog) MessageLogRun(msg_log *log.Logger) {
	for {
		select {
		case v := <-logger.MsgChan: msg_log.Printf("%s: %s\n", v.User, v.Message)
		case v := <-logger.NotifChan: msg_log.Printf("<Notification> %s\n", v.NotifBody)
		}
	}
}

func initLog() {
	logfile, err := os.OpenFile(LOG_FILE, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Error opening logfile: %v", err)
	}

	log.SetOutput(logfile)

	msg_logfile, err := os.OpenFile(MSG_LOG_FILE, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Error opening message log: %v", err)
	}

	msg_log := log.New(msg_logfile, "", log.LstdFlags)

	go msg_logger.MessageLogRun(msg_log)
}

func main() {
	initLog()
	log.Println("Starting MoeChat!\n")

	go h.run()

	http.HandleFunc("/", handler)
	http.HandleFunc("/chat", chatHandler)
	http.HandleFunc("/users", usersHandler)
	log.Fatal(http.ListenAndServe(":80", nil))
}
