package commands

import (
	"fmt"

	"github.com/daniel13112001/scout/cli"
)

func Help(args cli.ParsedArgs) error {

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
	return nil

}
