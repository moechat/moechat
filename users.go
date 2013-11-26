package main

/*import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)*/

type User struct {
	Name string `json:"username"`
	Email string `json:"email"`
	Id int64 `json:"id"`

	connections map[*connection]bool
}

var usersById map[int64]*User = make(map[int64]*User)
//var userDb sql.DB

func getUser(id int64) *User {
	return usersById[id]
}
