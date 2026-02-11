package main

import (
	"fmt"
	_ `log`
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go-util <command> [args...]")
		os.Exit(1)
	}


}
