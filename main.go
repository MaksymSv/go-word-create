package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Word Document Generator")
	fmt.Println("\nAvailable commands:")
	fmt.Println("  1. Generate a document (saves to file):")
	fmt.Println("     go run cmd/generate/main.go")
	fmt.Println("\n  2. Start HTTP server (serves document via HTTP):")
	fmt.Println("     go run cmd/server/main.go")
	fmt.Println("\nRun the desired command from the project root directory.")
	os.Exit(0)
}
