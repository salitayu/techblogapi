package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"techblogapi/auth"
	"techblogapi/models"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// Make models.BlogModel the dependency in Env
type Env struct {
	blog models.BlogModel
}

func main() {
	err := godotenv.Load("local.env")
	if err != nil {
		log.Fatalf("An error occured. Err: %s", err)
	}
	host := os.Getenv("host")
	port := os.Getenv("port")
	user := os.Getenv("user")
	pass := os.Getenv("pass")
	dbname := os.Getenv("db")

	// Initialize connection pool
	newport, err := strconv.Atoi(port)
	if err != nil {
		log.Fatal(err)
	}
	conn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, newport, user, pass, dbname)
	db, err := sql.Open("postgres", conn)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize Env with models.BlogModel that wraps connection pool
	env := &Env{
		blog: models.BlogModel{DB: db},
	}

	http.HandleFunc("/", Handle)
	http.HandleFunc("/register", env.Register)
	http.HandleFunc("/login", env.Login)
	http.HandleFunc("/categories", env.GetCategories)
	http.HandleFunc("/posts", env.GetPosts)
	http.HandleFunc("/logout", Logout)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func Handle(w http.ResponseWriter, r *http.Request) {
	loggedIn := auth.CheckSession(w, r)
	if loggedIn != http.StatusOK {
		return
	}
	var name string = "Sally"
	fmt.Fprintf(w, "Hi, my name is %s. Welcome to my tech blog :)", name)
}

func (env *Env) GetCategories(w http.ResponseWriter, r *http.Request) {
	loggedIn := auth.CheckSession(w, r)
	if loggedIn != http.StatusOK {
		return
	}
	// Execute the SQL query by calling the AllCategoriesMethod() from env.blog
	categories, err := env.blog.AllCategories()
	if err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	for i, category := range categories {
		fmt.Fprintf(w, "%v, %v, %s", i, category.CategoryID, category.CategoryName)
	}
}

func (env *Env) GetPosts(w http.ResponseWriter, r *http.Request) {
	loggedIn := auth.CheckSession(w, r)
	if loggedIn != http.StatusOK {
		return
	}
	posts, err := env.blog.AllPosts()
	if err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	for i, post := range posts {
		fmt.Fprintf(w, "%v, %s", i, post.PostMessage)
	}
}

func (env *Env) Register(w http.ResponseWriter, r *http.Request) {
	// Get User Details from JSON
	var u models.User
	err := json.NewDecoder(r.Body).Decode(&u)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	userAdded, err := env.blog.Register(u)
	fmt.Fprintf(w, "%t", userAdded)
	fmt.Fprintf(w, "Added user %s with password %s", u.Username, u.Password)
	if err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
}

func (env *Env) Login(w http.ResponseWriter, r *http.Request) {
	var lc auth.LoginCredentials
	err := json.NewDecoder(r.Body).Decode(&lc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	loginSuccessful, err := env.blog.Login(lc)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if loginSuccessful {
		auth.CreateSession(w, lc)
		fmt.Fprintf(w, "Logged in %t", loginSuccessful)
	} else {
		http.SetCookie(w, &http.Cookie{
			Name:    "session_token",
			Value:   "",
			Expires: time.Now(),
		})
		fmt.Fprintf(w, "Invalid Credentials")
	}
}

func Refresh(w http.ResponseWriter, r *http.Request) {
	auth.RefreshSession(w, r)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	auth.RemoveSession(w, r)
}
