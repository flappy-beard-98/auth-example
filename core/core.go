package core

import (
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
)

// GetOrCreateAuthSessionID checks if auth_session_id cookie exists in the request and if not, creates a new auth_session_id cookie,
// returns the auth_session_id cookie value and true if the auth_session_id cookie does not exist
func GetOrCreateAuthSessionID(w http.ResponseWriter, r *http.Request) (string, bool) {
	sessionCookie, err := r.Cookie("auth_session_id")

	var sessionID string
	var created bool

	if err != nil || sessionCookie.Value == "" {
		sessionID = uuid.NewString()
		created = true
	} else {
		sessionID = sessionCookie.Value
		created = false
	}

	http.SetCookie(w, &http.Cookie{
		Name:  "auth_session_id",
		Value: sessionID,
		Path:  "/",
	})
	return sessionID, created
}

// recoverer is a middleware that recovers from panics
func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// cors is a middleware that adds CORS headers
func cors(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == "OPTIONS" {
			return
		}
		h.ServeHTTP(w, r)
	})
}

// logger is a middleware that logs the requests
func logger(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("HTTP %s %s", r.Method, r.RequestURI)
		h.ServeHTTP(w, r)
	})
}

// ServeHttp starts the http server with CORS and request logging
func ServeHttp() {
	loggedHandler := recoverer(logger(cors(http.DefaultServeMux)))

	log.Println("Server started on :8080")
	log.Panic(http.ListenAndServe(":8080", loggedHandler))
}
