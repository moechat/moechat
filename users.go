package main

type User struct {
	Name string `json:"username"`
	Email string `json:"email"`
	ID int `json:"id"`

	connections map[*connection]bool
}

var usersByID map[int]*User = make(map[int]*User)

func getUser(id int) *User {
	return usersByID[id]
}
