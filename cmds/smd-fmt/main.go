package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/gfrey/smutje/parser"
	"github.com/pkg/errors"
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
			return errors.Errorf("file %q does not exist", file)
		case err != nil:
			return errors.Wrap(err, "failed to stat smd file")
		}

		astNode, err := parser.Parse(file)
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(file, []byte(astNode.String()), stat.Mode().Perm())
		if err != nil {
			return errors.Wrap(err, "failed to write file")
		}
	}
	return nil
}
