package main

import (
	"fmt"
	"os"

	"github.com/daniel13112001/scout/commands"
)

type Command func([]string) error

func main() {

	args := os.Args

	commandMap := map[string]Command{
		"find":  commands.Find,
		"index": commands.Index,
	}

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

	// args[1] not args[2] because commands expect the name of the command as the first parameter.
	// this is to keep similar behaviour to os.Args where args[0] is the file name and args[1]: are the relevant params
	cmd, ok := commandMap[args[1]]

	if !ok {
		fmt.Printf("Unrecognized command. %v is not a valid command\n", args[1])
		return
	}

	err := cmd(args[1:])

	if err != nil {
		fmt.Println(err)
	}

}
