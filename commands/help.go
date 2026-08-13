package commands

import (
	"fmt"
	"context"

	"github.com/daniel13112001/scout/cli"
	"github.com/daniel13112001/scout/app"
)

func Help(ctx context.Context, args cli.ParsedArgs, deps app.Dependencies) error {

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
