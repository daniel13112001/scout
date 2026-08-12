package commands

import (
	"flag"
	"fmt"
)

func Find(args []string) error {

	setupFindCommandFlags()
	flag.Parse()

	fmt.Printf("Find command executing...\n")

	return nil

}

func setupFindCommandFlags() {

}
