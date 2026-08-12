package commands

import (
	"flag"
	"fmt"
)

func Index(args []string) error {

	setupIndexCommandFlags()
	flag.Parse()
	fmt.Printf("Index command executing...\n")
	return nil

}

func setupIndexCommandFlags() {

}
