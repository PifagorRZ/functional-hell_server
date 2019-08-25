package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/ws", handleConnections)
	go handleMessages()
	fs := http.FileServer(http.Dir("static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	// http.Handle("/chat", c.Handler(server))
	http.HandleFunc("/quest", questHandler)
	http.HandleFunc("/", notFoundHandler)
	http.ListenAndServe(":8080", nil)
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "404 Not Found")
}
