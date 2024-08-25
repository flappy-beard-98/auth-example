package main

import (
	"auth-example/core"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
)

var (
	clients  = sync.Map{}
	users    = sync.Map{}
	sessions = sync.Map{}
)

type Client struct {
	ID string `json:"id"`
}

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginOtp struct {
	Login string `json:"login"`
	Otp   string `json:"otp"`
}

func main() {
	initData()

	http.HandleFunc("/oauth/authorize", authorize)
	http.HandleFunc("/oauth/token", token)

	core.ServeHttp()
}

func initData() {
	clientId := os.Getenv("CLIENT_ID")
	userId := os.Getenv("USER")
	password := os.Getenv("PASSWORD")
	clients.Store(clientId, Client{ID: clientId})
	users.Store(userId, User{ID: userId, Username: userId, Password: password})
}

// authorize handles the OAuth authorization request, redirecting to auth-frontend
// if the user is not authenticated or returning a login and OTP if the user is authenticated
func authorize(w http.ResponseWriter, r *http.Request) {
	clientID := r.FormValue("client_id")
	redirectURI := r.FormValue("redirect_uri")

	if clientID == "" || redirectURI == "" {
		log.Panic("invalid_request")
	}

	_, ok := clients.Load(clientID)
	if !ok {
		log.Panic("unauthorized_client")
	}

	loginOtp, authenticated := authenticated(r)

	log.Println("check authenticated", loginOtp.Login, loginOtp.Otp, authenticated)

	if !authenticated {
		newURL := fmt.Sprintf("%s/?%s", os.Getenv("AUTH_FRONTEND"), r.URL.Query().Encode())
		http.Redirect(w, r, newURL, http.StatusFound)
		return
	}

	var redirectURL string

	switch r.FormValue("response_type") {
	case "game_auth":
		redirectURL = fmt.Sprintf("%s?login=%s&otp=%s&state=%s", redirectURI, loginOtp.Login, loginOtp.Otp, r.FormValue("state"))
	// other grant_type cases ignored in this example...
	default:
		log.Panic("unsupported_response_type")
		return
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// authenticated checks if the current user session is authenticated by validating the session cookie
func authenticated(r *http.Request) (LoginOtp, bool) {
	sessionCookie, err := r.Cookie("auth_session_id")

	if err != nil {
		return LoginOtp{}, false
	}

	log.Print(sessionCookie.Value)
	log.Printf("sessions a %v", sessions)
	loginOtp, ok := sessions.Load(sessionCookie.Value)
	if !ok {
		log.Print("not ok")
		return LoginOtp{}, false
	}

	if r, ok := loginOtp.(LoginOtp); ok {
		return r, true
	}

	return LoginOtp{}, false
}

// token handles the OAuth token request, delegating to the appropriate grant type handler
func token(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Panic("unable to parse form")
	}

	switch r.FormValue("grant_type") {
	case "login_otp":
		grantLoginOtp(w, r)
	// other grant_type cases ignored in this example...
	default:
		log.Panic("unsupported_grant_type")
	}
}

// grantLoginOtp processes the login_otp grant type, validating user credentials
// and returning a login and OTP in the response
func grantLoginOtp(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	clientID := r.FormValue("client_id")

	_, ok := clients.Load(clientID)
	if !ok {
		log.Panic("client not found")
	}

	userValue, ok := users.Load(username)
	if !ok {
		log.Panicf("user not found %s", username)
	}
	user := userValue.(User)
	if user.Password != password {
		log.Panic("wrong password")
	}

	response := LoginOtp{
		Login: fmt.Sprintf("lgn:%s", username),
		Otp:   fmt.Sprintf("otp:%d", rand.Int()),
	}

	sessionID, _ := core.GetOrCreateAuthSessionID(w, r)
	sessions.Store(sessionID, response)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
