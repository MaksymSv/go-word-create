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
	fmt.Println("\n  3. Get issues from Jira sprint:")
	fmt.Println("     go run cmd/get-from-jira/main.go -sprint=\"Sprint 16\"")
	fmt.Println("     (Configure Jira connection details in .env file)")
	fmt.Println("\nRun the desired command from the project root directory.")
	os.Exit(0)
}
