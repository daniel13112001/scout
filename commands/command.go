package commands

import (
	"context"

	"github.com/daniel13112001/scout/app"
	"github.com/daniel13112001/scout/cli"
)

type Command func(context.Context, cli.ParsedArgs, app.Dependencies) error
