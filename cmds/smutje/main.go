package main

import (
	"fmt"
	"os"

	"github.com/gfrey/smutje"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	}
}

func run() error {
	tgt, err := smutje.ReadFile(os.Args[1])
	if err != nil {
		return err
	}

	return smutje.Provision(tgt)
}
