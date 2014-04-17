package main

import (
	"github.com/go-martini/martini"
	// "github.com/russross/blackfriday"
	"database/sql"
	_ "github.com/lib/pq"
	"net/http"
)

func PanicIf(e error) {
	if e != nil {
		panic(e)
	}
}

func setupDB() *sql.DB {
	db, e := sql.Open("postgres", "dbname=Blog username=postgres password=postgres sslmode=disable")
	PanicIf(e)
}

func main() {
	m := martini.Classic()

	m.Get("/", func(rw http.ResponseWriter, r *http.Request) string {
		return "Hello world!"
	})

	m.Run()
}
