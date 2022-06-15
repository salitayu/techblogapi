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
		Password: "",
		DB:       0,
	})
	_, err := redisClient.Ping().Result()
	if err != nil {
		return nil, err
	}
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

func (rc *RedisClient) CheckSession(w http.ResponseWriter, r *http.Request) int {
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
	sessionTokenRedis, err := rc.Connection.Get(sessionToken).Result()
	jsonMap := Session{}
	json.Unmarshal([]byte(sessionTokenRedis), &jsonMap)
	if err != nil {
		return http.StatusUnauthorized
	}

	// Delete session token if expired
	if jsonMap.isExpired() {
		rc.Connection.Del(sessionToken)
		return http.StatusUnauthorized
	}

	return http.StatusOK
}

func (rc *RedisClient) CreateSession(w http.ResponseWriter, lc LoginCredentials) {
	// Create new random session token using uuid
	sessionToken := uuid.NewString()
	expiresAt := time.Now().Add(3600 * time.Second)

	// Setting token in Redis
	json, err := json.Marshal(Session{Username: lc.Username, Expiry: expiresAt})
	rc.Connection.Set(sessionToken, json, 0).Err()
	if err != nil {
		fmt.Println(err)
	}

	_, err = rc.Connection.Get(sessionToken).Result()
	if err != nil {
		fmt.Println(err)
	}

	// Set the client cookie for "session_token" as the session token generated and expiry of 120s
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   sessionToken,
		Expires: expiresAt,
	})
}

func (rc *RedisClient) RefreshSession(w http.ResponseWriter, r *http.Request) {
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

	_, err = rc.Connection.Get(sessionToken).Result()
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
	}

	// Create new session for current user if session is valid
	newSessionToken := uuid.NewString()
	expiresAt := time.Now().Add(3600 * time.Second)

	newSessionTokenString, err := json.Marshal(Session{Username: newSessionToken, Expiry: expiresAt})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	rc.Connection.Set(newSessionToken, newSessionTokenString, 0).Err()

	// Delete previous session
	rc.Connection.Del(sessionToken)

	// Set new token as user's session_token cookie
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   newSessionToken,
		Expires: time.Now().Add(3600 * time.Second),
	})
}

func (rc *RedisClient) RemoveSession(w http.ResponseWriter, r *http.Request) {
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

	// Removing Session from Redis
	rc.Connection.Del(sessionToken).Err()

	// Remove Cookie
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   "",
		Expires: time.Now(),
	})
}
