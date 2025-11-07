package main

import (
	"fmt"
	"log"
	"net/http"

	"go-word-create/internal/server"
)

func main() {
	// Create a new handler
	handler := server.NewHandler()

	// Register the route
	http.HandleFunc("/get-doc", handler.GetDocument)

	// Start the server
	port := 80
	fmt.Printf("Server starting on http://localhost:%d/get-doc\n", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
