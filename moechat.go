package main

import (
	"fmt"
	"net/http"
	"log"
	"os"
	"strings"
)

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

func main() {
	fmt.Printf("Starting MoeChat!\n")
	go h.run()
	http.HandleFunc("/", handler)
	http.HandleFunc("/chat", chatHandler)
	http.ListenAndServe(":80", nil)
}
