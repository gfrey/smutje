package smutje

import (
	"fmt"
	"path/filepath"

	"github.com/gfrey/smutje/logger"
	"github.com/gfrey/smutje/parser"
)

func ReadFile(filename string) (*smResource, error) {
	astN, err := parser.Parse(filename)
	if err != nil {
		return nil, err
	}

	return convertToTarget(filepath.Dir(filename), astN)
}

func convertToTarget(path string, astN *parser.AstNode) (*smResource, error) {
	switch astN.Type {
	case parser.AstResource:
		return newResource(path, astN)
	case parser.AstTemplate:
		return nil, fmt.Errorf("can't handle templates directly, use the include mechanism!")
	default:
		return nil, fmt.Errorf("unexpected node seen: %s", astN.Type)
	}
}

func Provision(res *smResource) error {
	l := logger.New()
	if err := res.Prepare(l); err != nil {
		return err
	}

	if err := res.Generate(l); err != nil {
		return err
	}

	return res.Provision(l)
}
