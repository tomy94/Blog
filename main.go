package main

import (
	// "fmt"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"github.com/martini-contrib/sessions"
	// "github.com/russross/blackfriday"
	"code.google.com/p/go.crypto/bcrypt"
	"database/sql"
	_ "github.com/lib/pq"
	"net/http"
)

type Article struct {
	Title  string
	Author string
	Body   string
}

type User struct {
	Username string
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

	//Sessions
	store := sessions.NewCookieStore([]byte("thisIsTheSecret"))

	m.Map(SetupDB())
	m.Use(sessions.Sessions("BlogSession", store))
	m.Use(render.Renderer(render.Options{
		Layout: "layout",
	}))

	m.Get("/", ShowArticles)
	m.Get("/login", Login)
	m.Post("/authorize", PostLogin)
	m.Get("/logout", LogOut)
	m.Get("/register", Register)
	m.Post("/signup", SignUp)
	m.Get("/articles", ShowArticles)
	m.Get("/create", RequireLogin, NewArticle)
	m.Post("/article", CreateArticle)

	m.Run()
}

func SignUp(rw http.ResponseWriter, r *http.Request, db *sql.DB) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	passwordr := r.FormValue("passwordR")

	if password == passwordr {
		hashedPassword, e := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		PanicIf(e)

		_, e = db.Exec("INSERT INTO users (username, pwd) VALUES ($1, $2);", username, hashedPassword)

		http.Redirect(rw, r, "/login", http.StatusFound)
	}
}

func Register(ren render.Render) {
	ren.HTML(200, "register", nil)
}

func LogOut(s sessions.Session) string {
	s.Delete("userId")
	return "logged out"
}

func RequireLogin(rw http.ResponseWriter, req *http.Request, s sessions.Session, db *sql.DB, c martini.Context) {

	user := &User{}
	e := db.QueryRow(`SELECT username FROM users WHERE id=$1`, s.Get("userId")).Scan(&user.Username)
	if e != nil {
		http.Redirect(rw, req, "/login", http.StatusFound)
		return
	}

	//map the user to the context
	c.Map(user)
}

func Login(ren render.Render) {
	ren.HTML(200, "login", nil)
}

func PostLogin(req *http.Request, db *sql.DB, s sessions.Session, ren render.Render) {
	var id, pass string

	username, password := req.FormValue("username"), req.FormValue("password")
	e := db.QueryRow("SELECT id, pwd FROM users WHERE username=$1", username).Scan(&id, &pass)
	PanicIf(e)

	if bcrypt.CompareHashAndPassword([]byte(pass), []byte(password)) == nil {
		//set the userId in the session
		s.Set("userId", id)

		ren.Redirect("/articles")
	}

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
