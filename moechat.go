package main

import (
	"fmt"
	"net/http"
	"log"
	"os"
	"strings"
)

var CLIENT_VER = "0.7"
var LOG_FILE = "/var/log/moechat.log"
var MSG_LOG_FILE = "/root/messages.log"
var MSG_LOG os.File

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

func initLog() {
	logfile, err := os.OpenFile(LOG_FILE, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)

	if err != nil {
		log.Fatal("Error opening logfile: %v", err)
	}

	log.SetOutput(logfile)

	MSG_LOG, err := os.OpenFile(MSG_LOG_FILE, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)

	if err != nil {
		log.Printf("Error opening logfile: %v", err)
	}
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
