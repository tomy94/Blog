package main

import (
	"fmt"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"github.com/martini-contrib/sessions"
	// "github.com/russross/blackfriday"
	"code.google.com/p/go.crypto/bcrypt"
	"database/sql"
	_ "github.com/lib/pq"
	"net/http"
	"strconv"
	"strings"
)

var err ErrorMsg

type Article struct {
	Id           int
	Title        string
	Author       string
	Body         string
	IsAuthor     bool
	CommentCount int
	Comments     []Comment
	User
	ErrorMsg
}

type Comment struct {
	Id        int
	Author    string
	Body      string
	ArticleId int
	IsAuthor  bool
}

type User struct {
	Username string
}

type ErrorMsg struct {
	Message string
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
	// m.Get("", RequireLogin, EditArticle)

	m.Get("/deleteComment/:commentId", RequireLogin, DeleteComment)
	m.Post("/postComment/:articleId", PostComment)
	m.Post("/edit/:articleId", RequireLogin, EditArticle)
	m.Post("/save/:articleId", RequireLogin, SaveArticle)
	m.Post("/delete/:articleId", RequireLogin, DeleteArticle)
	m.Get("/open/:articleId", OpenArticle)
	m.Get("/login", Login)
	m.Post("/authorize", PostLogin)
	m.Get("/logout", LogOut)
	m.Get("/register", Register)
	m.Post("/signup", SignUp)
	m.Get("/articles", ShowArticles)
	m.Get("/create", RequireLogin, NewArticle)
	m.Post("/article", RequireLogin, CreateArticle)

	http.ListenAndServe(":3000", m)
	// m.Run()
}

func DeleteComment(rw http.ResponseWriter, r *http.Request, db *sql.DB) {
	var articleId string
	idFromUrl := strings.TrimPrefix(r.URL.Path, "/deleteComment/")
	e := db.QueryRow(`SELECT article FROM comments WHERE id=$1;`, idFromUrl).Scan(&articleId)
	fmt.Println(idFromUrl)
	PanicIf(e)
	_, e = db.Exec(`DELETE FROM comments WHERE id=$1;`, idFromUrl)
	PanicIf(e)

	redirectUrl := "/open/" + articleId
	fmt.Println(redirectUrl)
	http.Redirect(rw, r, redirectUrl, http.StatusFound)
}

func PostComment(rw http.ResponseWriter, r *http.Request, db *sql.DB, s sessions.Session) {
	comment := Comment{}
	db.QueryRow(`SELECT username FROM users WHERE id=$1`, s.Get("userId")).Scan(&comment.Author)
	comment.Body = r.FormValue("comment")
	comment.ArticleId, _ = strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/postComment/"))

	if comment.Author == "" {
		comment.Author = "Guest"
	}

	_, e := db.Exec(`INSERT INTO comments (author, body, article) VALUES ($1, $2, $3);`, comment.Author, comment.Body, comment.ArticleId)
	PanicIf(e)

	redirectPath := "/open/" + strconv.Itoa(comment.ArticleId)
	http.Redirect(rw, r, redirectPath, http.StatusFound)
}

func EditArticle(rw http.ResponseWriter, r *http.Request, db *sql.DB, ren render.Render) {
	a := Article{}

	idFromUrl := strings.TrimPrefix(r.URL.Path, "/edit/")
	db.QueryRow(`SELECT title, body FROM articles WHERE id=$1;`, idFromUrl).Scan(&a.Title, &a.Body)
	a.Id, _ = strconv.Atoi(idFromUrl)
	ren.HTML(200, "edit-article", a)
}

func DeleteArticle(rw http.ResponseWriter, r *http.Request, db *sql.DB) {
	idFromUrl := strings.TrimPrefix(r.URL.Path, "/delete/")
	_, e := db.Exec(`DELETE FROM articles WHERE id=$1;`, idFromUrl)
	PanicIf(e)
	_, e = db.Exec(`DELETE FROM comments WHERE article=$1;`, idFromUrl)
	PanicIf(e)
	http.Redirect(rw, r, "/articles", http.StatusFound)
}

func SaveArticle(rw http.ResponseWriter, r *http.Request, db *sql.DB) {
	a := Article{}
	idFromUrl := strings.TrimPrefix(r.URL.Path, "/save/")
	a.Title = r.FormValue("title")
	a.Body = r.FormValue("body")
	a.Id, _ = strconv.Atoi(idFromUrl)

	_, e := db.Exec(`UPDATE articles SET title = $1, body = $2 WHERE id = $3;`, a.Title, a.Body, a.Id)
	PanicIf(e)
	http.Redirect(rw, r, "/articles", http.StatusFound)
}

func OpenArticle(rw http.ResponseWriter, r *http.Request, db *sql.DB, ren render.Render, s sessions.Session) {
	var a Article

	username := getUserById(s, db)

	idFromUrl := strings.TrimPrefix(r.URL.Path, "/open/")
	db.QueryRow(`SELECT title, author, body FROM articles WHERE id=$1`, idFromUrl).Scan(&a.Title, &a.Author, &a.Body)

	if a.Author == "" {
		http.Redirect(rw, r, "/articles", http.StatusFound)
	}
	rows, e := db.Query(`SELECT * FROM comments WHERE article=$1 ORDER BY id DESC;`, idFromUrl)
	PanicIf(e)
	defer rows.Close()

	for rows.Next() {
		c := Comment{}
		e := rows.Scan(&c.Id, &c.Author, &c.Body, &c.ArticleId)
		PanicIf(e)

		if username == a.Author {
			c.IsAuthor = true
		} else {
			c.IsAuthor = false
		}
		a.Comments = append(a.Comments, c)
	}

	a.Id, _ = strconv.Atoi(idFromUrl)

	if a.Author == username {
		a.IsAuthor = true
	} else {
		a.IsAuthor = false
	}

	ren.HTML(200, "showarticle", a)

}

func SignUp(rw http.ResponseWriter, r *http.Request, db *sql.DB, ren render.Render) {
	var test string
	err.Message = ""
	username := r.FormValue("username")
	password := r.FormValue("password")
	passwordr := r.FormValue("passwordR")

	e := db.QueryRow(`SELECT username FROM users WHERE username=$1`, username).Scan(&test)
	fmt.Println(test)

	if e != nil {
		if password == passwordr {
			fmt.Println("hashing password...")
			hashedPassword, e := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			PanicIf(e)

			_, e = db.Exec(`INSERT INTO users (username, pwd) VALUES ($1, $2);`, username, hashedPassword)

			http.Redirect(rw, r, "/login", http.StatusFound)
		} else {
			err.Message = "Passwords do not match!"
			fmt.Println(err.Message)
			http.Redirect(rw, r, "/register", http.StatusFound)
		}
	} else {
		err.Message = "This username has already been taken!"
		fmt.Println(err.Message)
		http.Redirect(rw, r, "/register", http.StatusFound)
	}

}

func Register(ren render.Render) {
	if err != (ErrorMsg{}) {
		ren.HTML(200, "register", err)
		err.Message = ""
	} else {
		ren.HTML(200, "register", nil)
	}
}

func LogOut(rw http.ResponseWriter, r *http.Request, s sessions.Session) {
	s.Delete("userId")
	http.Redirect(rw, r, "/login", http.StatusFound)
}

func RequireLogin(rw http.ResponseWriter, r *http.Request, s sessions.Session, db *sql.DB, c martini.Context) {
	user := &User{}
	e := db.QueryRow(`SELECT username FROM users WHERE id=$1`, s.Get("userId")).Scan(&user.Username)
	if e != nil {
		http.Redirect(rw, r, "/login", http.StatusFound)
	}

	//map the user to the context
	c.Map(user)
}

func Login(ren render.Render) {
	if err != (ErrorMsg{}) {
		ren.HTML(200, "login", err)
		err.Message = ""
	} else {
		ren.HTML(200, "login", nil)
	}
}

func PostLogin(rw http.ResponseWriter, r *http.Request, db *sql.DB, s sessions.Session, ren render.Render) {
	var id, pass string

	username, password := r.FormValue("username"), r.FormValue("password")
	e := db.QueryRow("SELECT id, pwd FROM users WHERE username=$1", username).Scan(&id, &pass)
	if e == nil {
		if bcrypt.CompareHashAndPassword([]byte(pass), []byte(password)) == nil {
			//set the userId in the session
			s.Set("userId", id)

			ren.Redirect("/articles")
		} else {
			err.Message = "Wrong password!"
			http.Redirect(rw, r, "/login", http.StatusFound)
		}
	} else {
		err.Message = "There is no such user!"
		http.Redirect(rw, r, "/login", http.StatusFound)
	}

}

func NewArticle(ren render.Render) {
	ren.HTML(200, "create-article", nil)
}

func CreateArticle(ren render.Render, r *http.Request, db *sql.DB, s sessions.Session) {
	var username string
	db.QueryRow(`SELECT username FROM users WHERE id=$1;`, s.Get("userId")).Scan(&username)

	_, e := db.Exec(`INSERT INTO articles (title, author, body) VALUES ($1, $2, $3);`,
		r.FormValue("title"),
		username,
		r.FormValue("body"))
	PanicIf(e)

	ren.Redirect("/")
}

func getUserById(s sessions.Session, db *sql.DB) string {
	var username string
	db.QueryRow(`SELECT username FROM users WHERE id=$1`, s.Get("userId")).Scan(&username)
	return username
}

func ShowArticles(ren render.Render, r *http.Request, db *sql.DB, s sessions.Session) {
	var username string

	rows, e := db.Query(`SELECT id, title, author, body FROM articles ORDER BY id DESC;`)
	PanicIf(e)
	defer rows.Close()

	db.QueryRow(`SELECT username FROM users WHERE id=$1`, s.Get("userId")).Scan(&username)

	articles := []Article{}

	for rows.Next() {
		a := Article{}
		e := rows.Scan(&a.Id, &a.Title, &a.Author, &a.Body)
		PanicIf(e)

		temp := strings.SplitAfterN(a.Body, " ", 31)
		a.Body = ""

		for i := 0; i < len(temp)-1; i++ {
			a.Body += temp[i]
		}
		a.Body += "..."

		articles = append(articles, a)
	}

	ren.HTML(200, "articles", articles)
}
