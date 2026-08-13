package commands

import (
	"fmt"
	"strconv"

	"github.com/daniel13112001/scout/cli"
)

type FindCommandOptions struct {
	max      int
	restrict string
}

func Find(args cli.ParsedArgs) error {

	if len(args.Positional) == 0 {
		return fmt.Errorf("Find command requires a query to search for")
	}

	if len(args.Positional) > 1 {
		return fmt.Errorf("find accepts at most one query, got %d: %v", len(args.Positional), args.Positional)
	}

	query := args.Positional[0]

	max := 5

	if value, ok := args.Flags["max"]; ok {
		parsedMax, err := strconv.Atoi(value)

		if err != nil {
			return fmt.Errorf("invalid value %q for --max: expected an integer", value)
		}

		max = parsedMax
	}

	restrict := args.Flags["restrict"]

	findCommandParams := FindCommandOptions{max, restrict}

	search(query, findCommandParams)

	fmt.Printf("Find command executing...\n")

	return nil

}

func search(query string, optionalFlags FindCommandOptions) error {

	fmt.Printf("Searching for %v with the following params: max: %d, restrict to: %s", query, optionalFlags.max, optionalFlags.restrict)
	return nil
}
