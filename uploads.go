package main

import (
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
)

const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz1234567890"

var uploadKeys = make(map[string]bool)
var tmpImageName = 0

func keyTimer() {

}

func genUploadKey(uid int64) string {
	ret := idToStr(uid)
	ret += " "
	for i := 0; i < 30; i++ {
		ret += string(chars[rand.Intn(len(chars))])
	}
	uploadKeys[ret] = true

	return ret
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "You must post to this endpoint!", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	token, ok := query["token"]
	if !ok || token[0] == "" {
		w.Header().Add("WWW-Authenticate", "Token")
		http.Error(w, "You must provide a token!", http.StatusUnauthorized)
		return
	}
	if !uploadKeys[token[0]] {
		w.Header().Add("WWW-Authenticate", "Token")
		http.Error(w, "Token is invalid!", http.StatusUnauthorized)
		return
	}

	delete(uploadKeys, token[0])
	uidStr := token[0][:strings.Index(token[0], " ")]

	ulType, ok := query["type"]
	if !ok {
		http.Error(w, "A type argument is required!", http.StatusBadRequest)
	}
	switch ulType[0] {
	case "img":
		reader, err := r.MultipartReader()
		if err != nil {
			log.Println("Failed to open reader:", err)
			return
		}
		part, err := reader.NextPart()
		if err != nil {
			log.Println("Failed to get a part from the reader:", err)
			return
		}

		uid, err := strToId(uidStr)
		if err != nil {
			log.Println("Invalid UID in token:", err)
			return
		}

		fname := ""
		if uid < 0 {
			fname = strconv.Itoa(tmpImageName) + "-" + part.FileName()
			tmpImageName++
		} else {
			fname = path.Join(getUser(uid).Name, part.FileName())
		}

		f, err := os.Open(path.Join(config.ImageDir, "tmp", fname))
		if err != nil {
			log.Println("Failed to open file to write:", err)
			return
		}
		defer f.Close()

		_, err = io.Copy(f, part)
		if err != nil {
			log.Println("Failed to copy file:", err)
			return
		}
	default:
		http.Error(w, "Invalid type argument!", http.StatusBadRequest)
		log.Printf("Someone attempted to access upload?type=%s\n", ulType)
	}
}