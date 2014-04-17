package main

import (
	// "fmt"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	// "github.com/russross/blackfriday"
	"database/sql"
	_ "github.com/lib/pq"
	"net/http"
)

type Article struct {
	Title  string
	Author string
	Body   string
}

func PanicIf(e error) {
	if e != nil {
		panic(e)
	}
}

func SetupDB() *sql.DB {
	db, e := sql.Open("postgres", "user=postgres password=postgres host=localhost dbname=Blog sslmode=disable")
	PanicIf(e)
	return db
}

func main() {
	m := martini.Classic()
	m.Map(SetupDB())
	m.Use(render.Renderer(render.Options{
		Layout: "layout",
	}))

	m.Get("/", FirstPage)
	m.Post("/login", PostLogin)
	m.Get("/articles", ShowArticles)
	m.Get("/create", NewArticle)
	m.Post("/article", CreateArticle)

	m.Run()
}

func FirstPage(ren render.Render) {
	ren.HTML(200, "login", nil)
}

func PostLogin(req *http.Request, db *sql.DB) (int, string) {
	var id string

	username, password := req.FormValue("username"), req.FormValue("password")
	e := db.QueryRow("SELECT id FROM users WHERE username=$1 AND pwd=$2", username, password).Scan(&id)

	if e != nil {
		return 401, "Not Authorized"
	}

	return 200, "User id is: " + id
}

func NewArticle(ren render.Render) {
	ren.HTML(200, "create-article", nil)
}

func CreateArticle(ren render.Render, r *http.Request, db *sql.DB) {
	rows, e := db.Query(`INSERT INTO articles (title, author, body) VALUES ($1, $2, $3);`,
		r.FormValue("title"),
		r.FormValue("author"),
		r.FormValue("body"))
	PanicIf(e)
	defer rows.Close()

	ren.Redirect("/")
}

func ShowArticles(ren render.Render, r *http.Request, db *sql.DB) {
	rows, e := db.Query(`SELECT title, author, body FROM articles;`)
	PanicIf(e)
	defer rows.Close()

	articles := []Article{}

	for rows.Next() {
		a := Article{}
		e := rows.Scan(&a.Title, &a.Author, &a.Body)
		PanicIf(e)
		articles = append(articles, a)

	}

	ren.HTML(200, "articles", articles)
}
