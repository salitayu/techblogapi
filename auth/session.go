package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-redis/redis"
	"github.com/google/uuid"
)

type RedisClient struct {
	Connection *redis.Client
}

func ConnectRedis() (*redis.Client, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // No password set
		DB:       0,  // Use default DB
	})
	ping, err := redisClient.Ping().Result()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	fmt.Println("redis connection with message: ", ping)

	return redisClient, nil
}

// struct to store user session in redis
type Session struct {
	Username string
	Expiry   time.Time
}

func (s Session) isExpired() bool {
	return s.Expiry.Before(time.Now())
}

func (redisClient *RedisClient) CheckSession(w http.ResponseWriter, r *http.Request) int {
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

	// Check session token from cookie and redis
	sessionTokenValue, err := redisClient.Connection.Get("session_token").Result()
	jsonMap := Session{}
	json.Unmarshal([]byte(sessionTokenValue), &jsonMap)
	if err != nil {
		panic(err)
	}

	fmt.Println("session_token from redis is: ", sessionTokenValue)
	fmt.Println("session_token from cookie is: ", sessionToken)

	if sessionToken != jsonMap.Username {
		// Return unauthorized error if token is not in our sessionToken map
		return http.StatusUnauthorized
	}

	// Delete session token if expired
	if jsonMap.isExpired() {
		// delete(sessions, sessionToken)
		fmt.Fprintf(w, "%v", http.StatusUnauthorized)
		return http.StatusUnauthorized
	}

	return http.StatusOK
}

func (redisClient *RedisClient) CreateSession(w http.ResponseWriter, lc LoginCredentials) {
	// Create new random session token using uuid
	sessionToken := uuid.NewString()
	expiresAt := time.Now().Add(3600 * time.Second)

	// Setting token in Redis
	json, err := json.Marshal(Session{Username: sessionToken, Expiry: expiresAt})
	redisClient.Connection.Set("session_token", json, 0).Err()
	if err != nil {
		fmt.Println(err)
	}

	sessionTokenValue, err := redisClient.Connection.Get("session_token").Result()
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

func (redisClient *RedisClient) RefreshSession(w http.ResponseWriter, r *http.Request) {
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

	sessionTokenValue, err := redisClient.Connection.Get("session").Result()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	jsonMap := Session{}
	json.Unmarshal([]byte(sessionTokenValue), &jsonMap)
	if sessionToken != jsonMap.Username {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Create new session for current user if session is valid
	newSessionToken := uuid.NewString()
	expiresAt := time.Now().Add(3600 * time.Second)

	newSessionTokenString, err := json.Marshal(Session{Username: newSessionToken, Expiry: expiresAt})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	redisClient.Connection.Set("session_token", newSessionTokenString, 0).Err()

	// Delete previous session
	// delete(sessions, sessionToken)

	// Set new token as user's session_token cookie
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   newSessionToken,
		Expires: time.Now().Add(3600 * time.Second),
	})
}

func (redisClient *RedisClient) RemoveSession(w http.ResponseWriter, r *http.Request) {
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

	// TODO: remove sessions from redis
	fmt.Println("sessionToken to delete: ", sessionToken)
	// delete(sessions, sessionToken)

	// Remove Cookie
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   "",
		Expires: time.Now(),
	})
}
