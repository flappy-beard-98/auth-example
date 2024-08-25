package main

import (
	"auth-example/core"
	_ "embed"
	"html/template"
	"net/http"
	"os"
)

//go:embed index.html
var html string 

func main() {
	http.HandleFunc("/", index)

	core.ServeHttp()
}

// index shows the index.html page with authorization frontend
func index(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("index").Parse(html))

	data := struct {
		AuthBackendURL string
	}{
		AuthBackendURL: os.Getenv("AUTH_BACKEND_URL"),
	}

	err := tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
