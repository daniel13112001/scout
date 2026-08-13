package commands

import (
	"context"
	"fmt"

	"github.com/daniel13112001/scout/cli"
	"github.com/daniel13112001/scout/app"
)

func Config(ctx context.Context, args cli.ParsedArgs, deps app.Dependencies) error {

	fmt.Println("Scout config command invoked...")
	return nil

}
