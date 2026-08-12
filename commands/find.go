package commands

import (
	"flag"
	"fmt"
)

type FindCommandOptions struct {
	max      int
	restrict string
}

func Find(args []string) error {

	if len(args) < 2 {
		return fmt.Errorf("Find command requires a query to search for")
	}

	query := args[1]

	fs := flag.NewFlagSet("find", flag.ContinueOnError)

	max := fs.Int("max", 5, "the max number of matches to return. Default is 5")
	restrict := fs.String("restrict", "", "the directory to restrict the search to. Default is none, i.e global search")

	fs.Parse(args[2:])
	findCommandParams := FindCommandOptions{*max, *restrict}

	search(query, findCommandParams)

	fmt.Printf("Find command executing...\n")

	return nil

}

func search(query string, optionalFlags FindCommandOptions) error {

	fmt.Printf("Searching for %v with the following params: max: %d, restrict to: %s", query, optionalFlags.max, optionalFlags.restrict)
	return nil
}
