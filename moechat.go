package main

import (
	"fmt"
	"net/http"
	"os"
	"io/ioutil"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handling connection: " + r.URL.Path)
	name := "/srv/chat" + r.URL.Path
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
	fmt.Println("Error handling request: " + err.Error())
	if os.IsNotExist(err) {
		body, err := ioutil.ReadFile("/srv/chat/404.html")
		if err == nil {
			fmt.Fprintf(w, string(body))
		}
	}
}

func main() {
	fmt.Printf("Starting MoeChat!\n")
	go h.run()
	http.HandleFunc("/", handler)
	http.HandleFunc("/chat", chatHandler)
	http.ListenAndServe(":8080", nil)
}
