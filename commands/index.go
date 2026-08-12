package commands

import (
	"flag"
	"fmt"
	"path/filepath"
)

type indexCommandOptions struct {
	isRecursive bool
	extensions  string
}

func Index(args []string) error {

	if len(args) < 2 {
		return fmt.Errorf("Index command requires a folder path to index")
	}

	dir := args[1]

	fs := flag.NewFlagSet("index", flag.ContinueOnError)

	recurse := fs.Bool("recursive", true, "whether sub-directories should also be indexed. the default is true.")
	// TODO. Add support for extensions. What should the syntax be? comma separated list? Or multiple -extensions?
	// For now this allows just one extension
	ext := fs.String("extensions", "", "specify file extensions to restrict indexing to")

	fs.Parse(args[2:])
	optionalFlags := indexCommandOptions{*recurse, *ext}

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
