package main

import (
	"code.google.com/p/go.crypto/otr"
	"crypto/rand"
	"html/template"
	"net/http"
	"log"
	"os"
	"path"
	"strings"
)

var privKey *otr.PrivateKey = &otr.PrivateKey{}
var moechat struct{Version string}

func handler(w http.ResponseWriter, r *http.Request) {
	ip := strings.Split(r.RemoteAddr,":")[0]
	if ok := config.BlockedIPs[ip]; ok {
		return
	}

	log.Println("Handling connection from ip " + ip + " for: " + r.URL.Path)
	if r.URL.Path == "/" {
		name := path.Join(config.ServerRoot, "index.html")
		t, err := template.ParseFiles(name)
		if err != nil {
			errorHandler(w, r, err)
			return
		}
		err = t.Execute(w, moechat)
		if err != nil {
			errorHandler(w, r, err)
			return
		}
	} else {
		name := ""
		if strings.HasPrefix(r.URL.Path, "/uploads") {
			name = path.Join(config.UploadDir, r.URL.Path[8:])
		} else {
			name = path.Join(config.ServerRoot, r.URL.Path)
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
}

func errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	log.Println("Error handling request: " + err.Error())
	if os.IsNotExist(err) {
		name := path.Join(config.ServerRoot, "404.html")
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
	parseConf()

	moechat = struct{Version string}{config.Version}

	initLog()

	os.Mkdir(config.UploadDir, 0777)
	os.Mkdir(path.Join(config.UploadDir, "tmp"), 0777)

	privKey.Generate(rand.Reader)

	go h.run()

	http.HandleFunc("/", handler)
	http.HandleFunc("/chat", chatHandler)
	http.HandleFunc("/users", usersHandler)
	http.HandleFunc("/upload", uploadHandler)

	log.Println("Starting MoeChat!\n")
	log.Fatal(http.ListenAndServe(":80", nil))
}
