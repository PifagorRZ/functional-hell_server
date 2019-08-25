package main

import (
	"fmt"
	"net/http"
)

func chatHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Chat Handler")
}
