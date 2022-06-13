package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-redis/redis"
	"github.com/google/uuid"
)

func ConnectRedis() (*redis.Client, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // No password set
		DB:       0,  // Use default DB
	})
	_, err := redisClient.Ping().Result()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return redisClient, nil
}

// store user session in redis
var sessions = map[string]Session{}

type Session struct {
	Username string
	Expiry   time.Time
}

func (s Session) isExpired() bool {
	return s.Expiry.Before(time.Now())
}

func CheckSession(w http.ResponseWriter, r *http.Request) int {
	// Get session_token from request cookies
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			// check if cookie is not set
			return http.StatusUnauthorized
		}
		return http.StatusBadRequest
	}
	sessionToken := c.Value

	// Check session token from map
	// TODO: get session from redis here
	userSession, exists := sessions[sessionToken]
	if !exists {
		// Return unauthorized error if token is not in our sessionToken map
		return http.StatusUnauthorized
	}

	// Delete session token if expired
	if userSession.isExpired() {
		delete(sessions, sessionToken)
		fmt.Fprintf(w, "%v", http.StatusUnauthorized)
		return http.StatusUnauthorized
	}

	return http.StatusOK
}

func CreateSession(w http.ResponseWriter, lc LoginCredentials) {
	// Create new random session token using uuid
	sessionToken := uuid.NewString()
	expiresAt := time.Now().Add(120 * time.Second)

	// Set token, username, and expiry info in the session map
	sessions[sessionToken] = Session{
		Username: lc.Username,
		Expiry:   expiresAt,
	}

	// Setting token in Redis
	client, err := ConnectRedis()
	if err != nil {
		fmt.Printf("error connecting to redis %s", client)
		// return http.StatusInternalServerError
	}
	json, err := json.Marshal(Session{Username: lc.Username, Expiry: expiresAt})
	client.Set("session_token", json, 0).Err()
	if err != nil {
		fmt.Println(err)
	}

	sessionTokenValue, err := client.Get("session_token").Result()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("session_token from redis is: ", sessionTokenValue)

	// Set the client cookie for "session_token" as the session token generated and expiry of 120s
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   sessionToken,
		Expires: expiresAt,
	})
}

func RefreshSession(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sessionToken := c.Value

	userSession, exists := sessions[sessionToken]
	if !exists {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if userSession.isExpired() {
		delete(sessions, sessionToken)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Create new session for current user if session is valid
	newSessionToken := uuid.NewString()
	expiresAt := time.Now().Add(120 * time.Second)

	// Set new session token in map
	sessions[newSessionToken] = Session{
		Username: userSession.Username,
		Expiry:   expiresAt,
	}

	// Delete previous session
	delete(sessions, sessionToken)

	// Set new token as user's session_token cookie
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   newSessionToken,
		Expires: time.Now().Add(120 * time.Second),
	})
}

func RemoveSession(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			// Return unauthorized if cookie is not set
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// For any other type of error, return a bad request status
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sessionToken := c.Value

	// Remove user sessions from session map
	delete(sessions, sessionToken)

	// Remove Cookie
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   "",
		Expires: time.Now(),
	})
}
