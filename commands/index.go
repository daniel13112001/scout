package commands

import (
	"fmt"
	"path/filepath"

	"github.com/daniel13112001/scout/cli"
)

type indexCommandOptions struct {
	isRecursive bool
	extensions  string
}

func Index(args cli.ParsedArgs) error {

	if len(args.Positional) == 0 {
		return fmt.Errorf("Index command requires a folder path to index")
	}

	if len(args.Positional) > 1 {
		return fmt.Errorf("index accepts at most one path, got %d: %v", len(args.Positional), args.Positional)
	}

	dir := args.Positional[0]

	recurse, err := boolFlag(args, "recursive", true)

	if err != nil {
		return err
	}

	// TODO. Add support for extensions. What should the syntax be? comma separated list? Or multiple -extensions?
	// For now this allows just one extension
	ext := args.Flags["extensions"]

	optionalFlags := indexCommandOptions{recurse, ext}

	absPath, err := filepath.Abs(dir)

	if err != nil {
		return fmt.Errorf("Unable to resolve path %v. Error: %v", dir, err.Error())
	}

	index(absPath, optionalFlags)

	return nil

}

func index(path string, optionalFlags indexCommandOptions) error {

	fmt.Printf("Indexing %v with the following flags: isRecursive: %v, extensions %v", path, optionalFlags.isRecursive, optionalFlags.extensions)

	return nil

}
