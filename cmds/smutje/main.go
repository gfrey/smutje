package main

import (
	"fmt"
	"os"

	"github.com/gfrey/smutje"
	"github.com/pkg/errors"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}
}

func run() error {
	if len(os.Args) != 2 {
		return errors.Errorf("usage: %s <smt-file>", os.Args[0])
	}

	tgt, err := smutje.ReadFile(os.Args[1])
	if err != nil {
		return err
	}

	return smutje.Provision(tgt)
}
