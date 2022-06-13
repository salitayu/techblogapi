package models

import (
	"database/sql"
	"techblogapi/auth"
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
	CategoryID   int64
	CategoryName string
}

type Post struct {
	PostID      int64
	PostMessage string
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

func (m BlogModel) AllPosts() ([]Post, error) {
	rows, err := m.DB.Query("SELECT message FROM post;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []Post
	for rows.Next() {
		var post Post
		err := rows.Scan(&post.PostID, &post.PostMessage)
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
