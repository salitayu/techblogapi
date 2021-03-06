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

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// Make models.BlogModel the dependency in Env
type Env struct {
	blog  models.BlogModel
	cache auth.RedisClient
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

	redisConn, err := auth.ConnectRedis()
	if err != nil {
		panic(err)
	}

	// Initialize Env with models.BlogModel that wraps connection pool
	env := &Env{
		blog:  models.BlogModel{DB: db},
		cache: auth.RedisClient{Conn: redisConn},
	}

	r := mux.NewRouter()
	// r.Use(contentTypeApplicationJsonMiddleware)

	r.HandleFunc("/", env.Handle).Methods("GET")
	r.HandleFunc("/register", env.Register).Methods("POST")
	r.HandleFunc("/login", env.Login).Methods("POST")
	r.HandleFunc("/checkSession", env.Handle).Methods("POST")

	r.HandleFunc("/categories", env.GetCategories).Methods("GET")
	r.HandleFunc("/categories/id/{id}", env.GetCategoryByID).Methods("GET")
	r.HandleFunc("/categories/name/{name}", env.GetIDForCategory).Methods("GET")
	r.HandleFunc("/category", env.InsertCategory).Methods("POST")
	// r.HandleFunc("/categories", env.BulkInsertCategories).Methods("POST")
	r.HandleFunc("/category/{id}", env.EditCategory).Methods("PUT")
	r.HandleFunc("/category/{id}", env.DeleteCategory).Methods("DELETE")

	r.HandleFunc("/posts", env.GetPosts).Methods("GET")
	r.HandleFunc("/posts/category/{id}", env.GetPostsByCategoryId).Methods("GET")
	r.HandleFunc("/posts/category/slug/{slug}", env.GetPostsByCategorySlug).Methods("GET")
	r.HandleFunc("/post/id/{id}", env.GetPostById).Methods("GET")
	r.HandleFunc("/post/slug/{slug}", env.GetPostBySlug).Methods("GET")
	r.HandleFunc("/post", env.InsertPost).Methods("POST")
	// r.HandleFunc("/posts", env.BulkInsertPosts).Methods("POST")
	r.HandleFunc("/post/{id}", env.EditPost).Methods("PUT")
	r.HandleFunc("/post/{id}", env.DeletePost).Methods("DELETE")

	r.HandleFunc("/comments", env.GetComments).Methods("GET")
	// r.HandleFunc("/comments/post/{postid}", env.GetCommentsByPostId).Methods("GET")
	// r.HandleFunc("/comments/user/{userid}", env.GetPostByUserId).Methods("GET")
	r.HandleFunc("/comment", env.InsertComment).Methods("POST")
	// r.HandleFunc("/comments/post/{id}", env.BulkInsertComments).Methods("POST")
	r.HandleFunc("/comment/{id}", env.EditComment).Methods("EDIT")
	r.HandleFunc("/comment/{id}", env.DeleteComment).Methods("DELETE")
	// r.HandleFunc("/comments/post/{id}", env.DeleteCommentsByPostId).Methods("DELETE")

	// r.HandleFunc("/images", env.GetImages).Methods("GET")
	// r.HandleFunc("/images/post/{postid}", env.GetImagesByPostId).Methods("GET")
	// r.HandleFunc("/image", env.InsertImage).Methods("POST")
	// r.HandleFunc("/images/post/{id}", env.BulkInsertImages).Methods("POST")
	// r.HandleFunc("/image/{id}", env.EditImage).Methods("EDIT")
	// r.HandleFunc("/image/{id}", env.DeleteImage).Methods("DELETE")
	// r.HandleFunc("/image/post/{id}", env.DeleteImageByPostId).Methods("DELETE")

	r.HandleFunc("/logout", env.Logout).Methods("POST")

	headersOk := handlers.AllowedHeaders([]string{"Content-Type", "Content-Length", "Accept", "Accept-Encoding", "X-Requested-With", "X-CSRF-Token", "Set-Cookie", "Authorization"})
	originsOk := handlers.AllowedOrigins([]string{"http://127.0.0.1:3000", "127.0.0.1:3000", "localhost:3000", "http://localhost:3000"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS", "DELETE"})
	allowCreds := handlers.AllowCredentials()
	exposedHeaders := handlers.ExposedHeaders([]string{"Set-Cookie"})

	// start server listen with error handling
	r.Use(contentTypeApplicationJsonMiddleware)
	log.Fatal(http.ListenAndServe(":8080", handlers.CORS(originsOk, headersOk, methodsOk, exposedHeaders, allowCreds)(r)))
	http.Handle("/", r)
}

func contentTypeApplicationJsonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func (env *Env) HandleCheck(w http.ResponseWriter, r *http.Request) int {
	loggedIn := env.cache.CheckSession(w, r)
	fmt.Println("logged in ", loggedIn)

	for _, c := range r.Cookies() {
		fmt.Println("allCookies", c)
	}

	if loggedIn != http.StatusOK {
		response := map[string]int{"Login returned code": loggedIn}
		json.NewEncoder(w).Encode(response)
		return loggedIn
	}
	return loggedIn
}

func (env *Env) Handle(w http.ResponseWriter, r *http.Request) {
	responseCode := env.HandleCheck(w, r)
	if responseCode != http.StatusOK {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"results": "logged in"})
}

func (env *Env) GetCategories(w http.ResponseWriter, r *http.Request) {
	// Execute the SQL query by calling the AllCategoriesMethod() from env.blog
	categories, err := env.blog.AllCategories()
	if err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	json.NewEncoder(w).Encode(map[string][]models.Category{"results": categories})
}

func (env *Env) GetCategoryByID(w http.ResponseWriter, r *http.Request) {
	// responseCode := env.HandleCheck(w, r)
	// if responseCode != http.StatusOK {
	// 	return
	// }
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		return
	}
	fmt.Println("id ", id)
	categoryName, err := env.blog.GetCatNameByID(id)
	if err != nil {
		fmt.Fprintf(w, "%s", err)
		return
	}
	fmt.Println("categoryByID ", categoryName)
	json.NewEncoder(w).Encode(map[string]string{"category_name": categoryName})
}

func (env *Env) GetIDForCategory(w http.ResponseWriter, r *http.Request) {
	// responseCode := env.HandleCheck(w, r)
	// if responseCode != http.StatusOK {
	// 	return
	// }
	vars := mux.Vars(r)
	name := vars["name"]
	id, err := env.blog.GetCatIDByName(name)
	if err != nil {
		return
	}
	json.NewEncoder(w).Encode(map[string]int{"category_id": id})
}

func (env *Env) InsertCategory(w http.ResponseWriter, r *http.Request) {
	// responseCode := env.HandleCheck(w, r)
	// if responseCode != http.StatusOK {
	// 	return
	// }
	var c models.Category
	err := json.NewDecoder(r.Body).Decode(&c)
	if err != nil {
		fmt.Fprintf(w, "%s", err)
		return
	}
	env.blog.AddCategory(c)
}

func (env *Env) EditCategory(w http.ResponseWriter, r *http.Request) {
	// responseCode := env.HandleCheck(w, r)
	// if responseCode != http.StatusOK {
	// 	return
	// }
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Category Id: %v\n", vars["id"])
	categoryId, err := strconv.Atoi(vars["id"])
	if err != nil {
		return
	}
	category := models.Category{}
	json.NewDecoder(r.Body).Decode(&category)
	env.blog.PutCategory(categoryId, category.CategoryName)
}

func (env *Env) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	// responseCode := env.HandleCheck(w, r)
	// if responseCode != http.StatusOK {
	// 	return
	// }
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Category Id: %v\n", vars["id"])
	categoryId, err := strconv.Atoi(vars["id"])
	if err != nil {
		return
	}
	env.blog.DeleteCategory(categoryId)
}

func (env *Env) GetPosts(w http.ResponseWriter, r *http.Request) {
	// responseCode := env.HandleCheck(w, r)
	// if responseCode != http.StatusOK {
	// 	return
	// }
	posts, err := env.blog.AllPosts()
	if err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	// fmt.Println("posts ", posts)
	json.NewEncoder(w).Encode(map[string][]models.Post{"results": posts})
}

func (env *Env) GetPostById(w http.ResponseWriter, r *http.Request) {
	// responseCode := env.HandleCheck(w, r)
	// if responseCode != http.StatusOK {
	// 	return
	// }
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		return
	}
	post, err := env.blog.PostById(id)
	if err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	json.NewEncoder(w).Encode(map[string][]models.Post{"results": post})
}

func (env *Env) GetPostBySlug(w http.ResponseWriter, r *http.Request) {
	// responseCode := env.HandleCheck(w, r)
	// if responseCode != http.StatusOK {
	// 	return
	// }
	vars := mux.Vars(r)
	slug := vars["slug"]
	post, err := env.blog.PostBySlug(slug)
	if err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	json.NewEncoder(w).Encode(map[string][]models.Post{"results": post})
}

func (env *Env) GetPostsByCategoryId(w http.ResponseWriter, r *http.Request) {
	// responseCode := env.HandleCheck(w, r)
	// if responseCode != http.StatusOK {
	// 	return
	// }
	vars := mux.Vars(r)
	categoryid, err := strconv.Atoi(vars["id"])
	if err != nil {
		return
	}
	posts, err := env.blog.AllPostsByCatID(categoryid)
	json.NewEncoder(w).Encode(map[string][]models.Post{"results": posts})
}

func (env *Env) GetPostsByCategorySlug(w http.ResponseWriter, r *http.Request) {
	// responseCode := env.HandleCheck(w, r)
	// if responseCode != http.StatusOK {
	// 	return
	// }
	vars := mux.Vars(r)
	categorySlug := vars["slug"]
	posts, err := env.blog.AllPostsByCatSlug(categorySlug)
	if err != nil {
		return
	}
	json.NewEncoder(w).Encode(map[string][]models.Post{"results": posts})
}

func (env *Env) InsertPost(w http.ResponseWriter, r *http.Request) {
	// responseCode := env.HandleCheck(w, r)
	// if responseCode != http.StatusOK {
	// 	return
	// }
	post := models.Post{}
	err := json.NewDecoder(r.Body).Decode(&post)
	fmt.Println("post to insert ", post)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	_, err = env.blog.AddPost(post)
	if err != nil {
		return
	}
}

func (env *Env) EditPost(w http.ResponseWriter, r *http.Request) {
	// responseCode := env.HandleCheck(w, r)
	// if responseCode != http.StatusOK {
	// 	return
	// }
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Post Id: %v\n", vars["id"])
	postid, err := strconv.Atoi(vars["id"])
	if err != nil {
		return
	}
	newpost := models.Post{}
	json.NewDecoder(r.Body).Decode(&newpost)
	env.blog.PutPost(postid, newpost)
}

func (env *Env) DeletePost(w http.ResponseWriter, r *http.Request) {
	// responseCode := env.HandleCheck(w, r)
	// if responseCode != http.StatusOK {
	// 	return
	// }
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Post Id: %v\n", vars["id"])
	postid, err := strconv.Atoi(vars["id"])
	if err != nil {
		return
	}
	env.blog.DelPost(postid)
}

func (env *Env) GetComments(w http.ResponseWriter, r *http.Request) {
	comments, err := env.blog.AllComments()
	if err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	json.NewEncoder(w).Encode(map[string][]models.Comment{"results": comments})
}

func (env *Env) InsertComment(w http.ResponseWriter, r *http.Request) {
	// responseCode := env.HandleCheck(w, r)
	// if responseCode != http.StatusOK {
	// 	return
	// }
	var c models.Comment
	err := json.NewDecoder(r.Body).Decode(&c)
	if err != nil {
		fmt.Fprintf(w, "%s", err)
		return
	}
	env.blog.AddComment(c)
}

func (env *Env) EditComment(w http.ResponseWriter, r *http.Request) {
	// responseCode := env.HandleCheck(w, r)
	// if responseCode != http.StatusOK {
	// 	return
	// }
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Post Id: %v\n", vars["id"])
	postid, err := strconv.Atoi(vars["id"])
	if err != nil {
		return
	}
	newcomment := models.Comment{}
	json.NewDecoder(r.Body).Decode(&newcomment)
	env.blog.PutComment(postid, newcomment)
}

func (env *Env) DeleteComment(w http.ResponseWriter, r *http.Request) {
	// responseCode := env.HandleCheck(w, r)
	// if responseCode != http.StatusOK {
	// 	return
	// }
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Comment Id: %v\n", vars["id"])
	commentid, err := strconv.Atoi(vars["id"])
	if err != nil {
		return
	}
	env.blog.DelComment(commentid)
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
		fmt.Println("lc")
		fmt.Println(lc)
		http.Error(w, err.Error(), http.StatusBadRequest)
		fmt.Fprintf(w, "Bad Request")
		return
	}
	loginSuccessful, err := env.blog.Login(lc)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		fmt.Fprintf(w, "Internal Server Error")
		return
	}

	if loginSuccessful {
		sessionToken := env.cache.CreateSession(w, lc)
		json.NewEncoder(w).Encode(map[string]string{"results": sessionToken})
	} else {
		http.SetCookie(w, &http.Cookie{
			Name:    "session_token",
			Value:   "",
			Expires: time.Now(),
		})
		fmt.Fprintf(w, "Invalid Credentials")
	}
}

func (env *Env) Refresh(w http.ResponseWriter, r *http.Request) {
	env.cache.RefreshSession(w, r)
}

func (env *Env) Logout(w http.ResponseWriter, r *http.Request) {
	responseCode := env.HandleCheck(w, r)
	if responseCode != http.StatusOK {
		return
	}
	env.cache.RemoveSession(w, r)
}
