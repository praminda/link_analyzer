package main

import (
	"log"
	"net/http"

	apphttp "github.com/praminda/link_analyzer/internal/http"
)

func main() {
	mux := apphttp.NewRouter()
	addr := ":8080"
	log.Printf("Server started on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
