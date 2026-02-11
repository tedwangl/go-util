package main

import (
	"fmt"
	"go-util/pkg/utils"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go-util <command> [args...]")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "version":
		fmt.Println("go-util v0.1.0")
	case "help":
		fmt.Println("go-util - A utility collection in Go")
		fmt.Println("Commands:")
		fmt.Println("  version - Show version information")
		fmt.Println("  help    - Show this help message")
		fmt.Println("  reverse <text> - Reverse the input text")
		fmt.Println("  upper <text> - Convert text to uppercase")
		fmt.Println("  lower <text> - Convert text to lowercase")
		fmt.Println("  word-count <text> - Count words in text")
	case "reverse":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go-util reverse <text>")
			os.Exit(1)
		}
		result := utils.Reverse(os.Args[2])
		fmt.Printf("Reversed: %s\n", result)
	case "upper":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go-util upper <text>")
			os.Exit(1)
		}
		result := utils.ToUpper(os.Args[2])
		fmt.Printf("Uppercase: %s\n", result)
	case "lower":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go-util lower <text>")
			os.Exit(1)
		}
		result := utils.ToLower(os.Args[2])
		fmt.Printf("Lowercase: %s\n", result)
	case "word-count":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go-util word-count <text>")
			os.Exit(1)
		}
		count := utils.CountWords(os.Args[2])
		fmt.Printf("Word count: %d\n", count)
	default:
		log.Printf("Unknown command: %s", command)
		os.Exit(1)
	}
}
