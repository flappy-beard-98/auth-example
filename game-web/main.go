package main

import (
	"auth-example/core"
	_ "embed"
	"html/template"
	"log"
	"net/http"
	"os"
)

//go:embed index.html
var html string

func main() {
	http.HandleFunc("/", index)
	core.ServeHttp()
}

// index shows the index.html page with login button
func index(w http.ResponseWriter, r *http.Request) {

	tmpl := template.Must(template.New("index").Parse(html))

	data := struct {
		GameBackendURL string
	}{
		GameBackendURL: os.Getenv("GAME_BACKEND_URL"),
	}

	err := tmpl.Execute(w, data)
	if err != nil {
		log.Panic(err)
	}
}
