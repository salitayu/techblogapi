package models

import (
	"database/sql"
	"fmt"
	"techblogapi/auth"
	"time"
)

// Create customer BlogModel type which wraps the sql.DB connection pool
type BlogModel struct {
	DB *sql.DB
}

type User struct {
	IsGuest     bool   `json:"is_guest" db:"is_guest"`
	IsSuperuser bool   `json:"is_superuser" db:"is_superuser"`
	Username    string `json:"username" db:"username"`
	FirstName   string `json:"firstname" db:"firstname"`
	LastName    string `json:"lastname" db:"lastname"`
	Email       string `json:"email" db:"email"`
	Password    string `json:"password" db:"password"`
}

type Category struct {
	CategoryID   int64  `json:"category_id,omitempty" db:"id"`
	CategoryName string `json:"category_name" db:"category_name"`
}

type Post struct {
	UserID     int64     `json:"user_id" db:"id"`
	CategoryID int64     `json:"category_id" db:"category_id"`
	PostID     int64     `json:"post_id" db:"post_id"`
	Message    string    `json:"message" db:"message"`
	Title      string    `json:"title" db:"title"`
	Excerpt    string    `json:"excerpt" db:"excerpt"`
	ReadTime   int64     `json:"read_time" db:"read_time"`
	DateTime   time.Time `json:"date_time" db:"datetime"`
}

type Comment struct {
	CommentID int64  `json:"comment_id,omitempty" db:"id"`
	UserID    int64  `json:"user_id" db:"user_id"`
	Message   string `json:"message" db:"message"`
	PostID    int64  `json:"post_id" db:"post_id"`
}

// Use a method on the custom BlogModel type to run the SQL query.
func (m BlogModel) AllCategories() ([]Category, error) {
	rows, err := m.DB.Query("SELECT * FROM category;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var categories []Category
	for rows.Next() {
		var category Category
		err := rows.Scan(&category.CategoryID, &category.CategoryName)
		if err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return categories, nil
}

func (m BlogModel) GetCatNameByID(id int) (string, error) {
	category := Category{}
	rows, err := m.DB.Query("SELECT category_name FROM category WHERE id = $1", id)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	for rows.Next() {
		err := rows.Scan(&category.CategoryName)
		if err != nil {
			fmt.Println(err)
			return "", err
		}
	}
	return category.CategoryName, nil
}

func (m BlogModel) GetCatIDByName(name string) (int, error) {
	id := -1
	rows, err := m.DB.Query("SELECT id FROM category WHERE category_name = $1", name)
	if err != nil {
		fmt.Println(err)
		return id, err
	}
	for rows.Next() {
		err := rows.Scan(&id)
		if err != nil {
			fmt.Println(err)
			return id, err
		}
	}
	return id, nil
}

func (m BlogModel) AllPosts() ([]Post, error) {
	rows, err := m.DB.Query("SELECT message FROM post;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []Post
	for rows.Next() {
		var post Post
		err := rows.Scan(&post.PostID, &post.Message)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return posts, nil
}

func (m BlogModel) Register(u User) (bool, error) {
	// Auth Params
	p := &auth.AuthParams{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}

	// Generate Hash for Password
	encodedHash, err := auth.GenerateFromPassword(u.Password, p)
	if err != nil {
		return false, err
	}
	_, err = m.DB.Exec("INSERT INTO users (is_guest, is_superuser, username, firstname, lastname, email, password) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		u.IsGuest,
		u.IsSuperuser,
		u.Username,
		u.FirstName,
		u.LastName,
		u.Email,
		encodedHash)

	if err != nil {
		return false, err
	}
	return true, nil
}

func (m BlogModel) Login(lc auth.LoginCredentials) (bool, error) {
	var password string
	row := m.DB.QueryRow("SELECT password FROM users WHERE username = $1", lc.Username)
	if err := row.Scan(&password); err != nil {
		if err == sql.ErrNoRows {
			return false, err
		}
		return false, err
	}
	validCreds, err := auth.ComparePasswordAndHash(lc.Password, password)
	if err != nil {
		return false, err
	}
	return validCreds, nil
}

func (m BlogModel) AddCategory(c Category) (bool, error) {
	_, err := m.DB.Exec("INSERT INTO category(category_name) VALUES($1)", c.CategoryName)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (m BlogModel) PutCategory(categoryId int, newCategoryName string) (bool, error) {
	fmt.Println("categoryId ", categoryId)
	fmt.Println("newCategoryName ", newCategoryName)
	_, err := m.DB.Exec("UPDATE category SET category_name = $1 WHERE id = $2", newCategoryName, categoryId)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (m BlogModel) DeleteCategory(categoryId int) (bool, error) {
	_, err := m.DB.Exec("DELETE FROM category WHERE id = $1", categoryId)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (m BlogModel) AddPost(p Post) (bool, error) {
	_, err := m.DB.Exec("INSERT INTO post(user_id, category_id, title, excerpt, read_time, datetime, message) VALUES($1, $2, $3, $4, $5, $6, $7)",
		p.UserID, p.CategoryID, p.Title, p.Excerpt, p.ReadTime, p.DateTime, p.Message)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (m BlogModel) PutPost(postid int, p Post) (bool, error) {
	_, err := m.DB.Exec("UPDATE post SET user_id = $1, category_id = $2, title = $3, excerpt = $4, read_time = $5, datetime = $6, message = $7",
		p.UserID, p.CategoryID, p.Title, p.Excerpt, p.ReadTime, p.DateTime, p.Message)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (m BlogModel) DelPost(postid int) (bool, error) {
	_, err := m.DB.Exec("DELETE FROM post WHERE id = $1", postid)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (m BlogModel) AllComments() ([]Comment, error) {
	rows, err := m.DB.Query("SELECT * FROM comment")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var comments []Comment
	for rows.Next() {
		var comment Comment
		err := rows.Scan(&comment.CommentID, &comment.UserID, &comment.PostID, &comment.Message)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return comments, nil
}

func (m BlogModel) AddComment(c Comment) (bool, error) {
	_, err := m.DB.Exec("INSERT INTO comment(user_id, post_id, message) VALUES($1, $2, $3)", c.UserID, c.PostID, c.Message)
	if err != nil {
		fmt.Println(err)
		return false, err
	}
	return true, nil
}

func (m BlogModel) PutComment(postid int, c Comment) (bool, error) {
	_, err := m.DB.Exec("UPDATE comment SET user_id = $1, message = $2, post_id = $3",
		c.UserID, c.PostID, c.Message)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (m BlogModel) DelComment(commentid int) (bool, error) {
	_, err := m.DB.Exec("DELETE FROM comment WHERE id = $1", commentid)
	if err != nil {
		return false, err
	}
	return true, nil
}
