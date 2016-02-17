package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/gfrey/smutje/parser"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	for _, file := range os.Args[1:] {
		stat, err := os.Lstat(file)
		switch {
		case os.IsNotExist(err):
			return fmt.Errorf("file %q does not exist", file)
		case err != nil:
			return err
		}

		astNode, err := parser.Parse(file)
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(file, []byte(astNode.String()), stat.Mode().Perm())
		if err != nil {
			return err
		}
	}
	return nil
}
