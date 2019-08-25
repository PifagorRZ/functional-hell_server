package main

import (
	"fmt"
	"net/http"
)

func main() {
	fs := http.FileServer(http.Dir("static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.HandleFunc("/chat", chatHandler)
	http.HandleFunc("/quest", questHandler)
	http.HandleFunc("/", notFoundHandler)
	http.ListenAndServe(":80", nil)
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "404 Not Found")
}
