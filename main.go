package main

import (
	"fmt"
	"os"
)

func main() {

	args := os.Args

	if len(args) < 2 {

		usage := `
		Usage:
		scout <command> [arguments]

		Commands:
		find       Search indexed files
		index      Index files
		sync       Synchronize the index
		help       Show help

		`
		fmt.Println(usage)

		return

	}

	switch args[1] {
	case "find", "f":
		fmt.Println("scout is searching for ...")
	case "index", "i":
		fmt.Println("scout is indexing files ...")
	case "sync", "s":
		fmt.Println("scout is syncing...")
	case "help", "h":
		fmt.Println("usage ...")
	default:
		fmt.Printf("Unrecognized command %v", args[0])

	}

}
