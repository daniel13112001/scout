package commands

import (
	"flag"
	"fmt"
)

type IndexCommandOptions struct {
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
	params := IndexCommandOptions{*recurse, *ext}

	index(dir, params)

	return nil

}

func index(dir string, optionalFlags IndexCommandOptions) error {

	fmt.Printf("Indexing %v with the following flags: isRecursive: %v, extensions %v", dir, optionalFlags.isRecursive, optionalFlags.extensions)

	return nil

}
