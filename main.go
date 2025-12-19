package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// TODO: Initialize router, middleware, handlers
	// This is a placeholder - will be implemented in Phase 6

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"healthy"}`)
	})

	log.Printf("Gateway starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

