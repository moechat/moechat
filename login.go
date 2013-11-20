package main

import (
	"net/http"
)

var users map[string]User = make(map[string]User)

type User struct {
	Name string `json:"username"`
	Email string `json:"email"`
	ID int `json:"id"`
}

/* Checks login info against database and return a token */
func loginHandler(w http.ResponseWriter, r *http.Request) {

	//token := generateToken()

	//users[token] = User{username, email}
}
