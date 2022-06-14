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

func ConnectRedis() (*RedisClient, error) {
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

	return &RedisClient{Connection: redisClient}, nil
}

// struct to store user session in redis
type Session struct {
	Username string
	Expiry   time.Time
}

func (s Session) isExpired() bool {
	return s.Expiry.Before(time.Now())
}

func CheckSession(w http.ResponseWriter, r *http.Request, redisClient *RedisClient) int {
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

func CreateSession(w http.ResponseWriter, lc LoginCredentials, redisClient *RedisClient) {
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

// func RefreshSession(w http.ResponseWriter, r *http.Request) {
// 	c, err := r.Cookie("session_token")
// 	if err != nil {
// 		if err == http.ErrNoCookie {
// 			w.WriteHeader(http.StatusUnauthorized)
// 			return
// 		}
// 		w.WriteHeader(http.StatusBadRequest)
// 		return
// 	}
// 	sessionToken := c.Value

// 	userSession, exists := sessions[sessionToken]
// 	if !exists {
// 		w.WriteHeader(http.StatusUnauthorized)
// 		return
// 	}
// 	if userSession.isExpired() {
// 		delete(sessions, sessionToken)
// 		w.WriteHeader(http.StatusUnauthorized)
// 		return
// 	}

// 	// Create new session for current user if session is valid
// 	newSessionToken := uuid.NewString()
// 	expiresAt := time.Now().Add(120 * time.Second)

// 	// Set new session token in map
// 	sessions[newSessionToken] = Session{
// 		Username: userSession.Username,
// 		Expiry:   expiresAt,
// 	}

// 	// Delete previous session
// 	delete(sessions, sessionToken)

// 	// Set new token as user's session_token cookie
// 	http.SetCookie(w, &http.Cookie{
// 		Name:    "session_token",
// 		Value:   newSessionToken,
// 		Expires: time.Now().Add(120 * time.Second),
// 	})
// }

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
