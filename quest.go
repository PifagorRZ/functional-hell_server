package main

import (
	"fmt"
	"net/http"
)

func questHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Quest Handler")
}
