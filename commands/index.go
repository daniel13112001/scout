package commands

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/daniel13112001/scout/app"
	"github.com/daniel13112001/scout/cli"
)

func Index(ctx context.Context, args cli.ParsedArgs, deps app.Dependencies) error {

	if len(args.Positional) == 0 {
		return fmt.Errorf("Index command requires a folder path to index")
	}

	if len(args.Positional) > 1 {
		return fmt.Errorf("index accepts at most one path, got %d: %v", len(args.Positional), args.Positional)
	}

	dir := args.Positional[0]

	recursive, err := boolFlag(args, "recursive", true)

	if err != nil {
		return err
	}

	absPath, err := filepath.Abs(dir)

	if err != nil {
		return fmt.Errorf("Unable to resolve path %v. Error: %v", dir, err.Error())
	}

	return deps.FileIndexer.IndexDirectory(absPath, recursive)

}
