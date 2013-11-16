package main

import (
	"fmt"
	"net/http"
	"os"
	"io/ioutil"
)

func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		body, err := ioutil.ReadFile("/srv/chat/index.html")
		if err != nil {
			errorHandler(w, r, err)
			return
		}
		fmt.Fprintf(w, string(body))
	} else {
		body, err := ioutil.ReadFile("/srv/chat" + r.URL.Path)
		if err != nil {
			errorHandler(w, r, err)
			return
		}
		fmt.Fprintf(w, string(body))
	}
}

func errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	if os.IsNotExist(err) {
		body, err := ioutil.ReadFile("/srv/chat/404.html")
		if err == nil {
			fmt.Fprintf(w, string(body))
			return
		}
	}
	fmt.Fprintf(w, err.Error())
}

func main() {
	fmt.Printf("Starting MoeChat!\n")
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
