package main

import (
	"auth-example/core"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
)

var (
	// sessions stores the sessions in memory
	sessions = sync.Map{}
	// client is the http client with disabled redirects
	client = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
)

// Session represents an authorization session
type Session struct {
	ID    string // ID is unique identifier of the session
	Login string // Login is the game login
	Otp   string // Otp is the one time password
}

func main() {

	http.HandleFunc("/auth", auth)
	http.HandleFunc("/callback", callback)
	http.HandleFunc("/session", session)

	core.ServeHttp()
}

// auth handles game-backend authorization
func auth(w http.ResponseWriter, r *http.Request) {
	// retrieves or creates an authorization session to store the login and OTP in the callback method
	// if the session is created, it is stored in the session store
	sessionID, created := core.GetOrCreateAuthSessionID(w, r)
	log.Println("check session", sessionID, created)
	if created {
		session := Session{ID: sessionID}
		sessions.Store(sessionID, session)
	}

	// send authorization request to auth-backend
	authURL := fmt.Sprintf(
		"%s/oauth/authorize?client_id=%s&response_type=game_auth&redirect_uri=%s/callback&state=%s",
		os.Getenv("AUTH_BACKEND_URL"), os.Getenv("CLIENT_ID"), os.Getenv("BACKEND_URL"), sessionID)

	req, _ := http.NewRequest("GET", authURL, nil)
	if !created {
		req.Header.Set("Cookie", fmt.Sprintf("auth_session_id=%s", sessionID))
	}

	resp, err := client.Do(req)

	if err != nil {
		log.Panic(err.Error())
	}
	defer resp.Body.Close()

	// the auth-backend should either redirect to the auth-frontend for authentication
	// or return a redirect with the login and OTP if the user is already authenticated.
	// if a redirect is required, the auth-frontend URL and session ID are returned in the response.
	if resp.StatusCode == http.StatusFound {
		location, err := resp.Location()
		if err != nil {
			log.Panic("Failed to retrieve redirect location")
		}
		response := map[string]string{
			"authFrontendUrl": location.String(),
			"sessionId":       sessionID,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Panic("Failed to redirect")
}

// callback handles the OAuth callback after user authentication,
// it extracts the state, login, and OTP from the query parameters,
// validates them, and then stores the session information
func callback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	login := r.URL.Query().Get("login")
	otp := r.URL.Query().Get("otp")

	if login == "" || otp == "" || state == "" {
		log.Panic("Invalid request")
	}

	sessions.Store(state, Session{ID: state, Login: login, Otp: otp})

	w.WriteHeader(http.StatusOK)
}

// session handles requests to retrieve session information based on session_id,
// it extracts the session_id from the query parameters and looks up the session in the session store,
func session(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		log.Panic("session_id required")
	}

	sessionValue, ok := sessions.Load(sessionID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	session := sessionValue.(Session)
	response := map[string]string{
		"login": session.Login,
		"otp":   session.Otp,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
