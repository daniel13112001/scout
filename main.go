package main

import (
	"fmt"
	"os"

	"github.com/daniel13112001/scout/cli"
	"github.com/daniel13112001/scout/commands"
)

type Command func(cli.ParsedArgs) error

func main() {

	args := os.Args

	commandMap := map[string]Command{
		"find":  commands.Find,
		"index": commands.Index,
		"help": commands.Help,
		"sync": commands.Sync,
		"config": commands.Config,
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

	cmd, ok := commandMap[args[1]]

	if !ok {
		fmt.Printf("Unrecognized command. %v is not a valid command\n", args[1])
		return
	}

	parsedArgs, err := cli.Parse(args[2:])

	if err != nil {
		fmt.Println(err)
		return
	}

	err = cmd(parsedArgs)

	if err != nil {
		fmt.Println(err)
	}

}
