package smutje

import (
	"path/filepath"

	"github.com/gfrey/glog"
	"github.com/gfrey/smutje/parser"
	"github.com/pkg/errors"
)

func ReadFile(filename string) (*Resource, error) {
	astN, err := parser.Parse(filename)
	if err != nil {
		return nil, err
	}

	return convertToTarget(filepath.Dir(filename), astN)
}

func convertToTarget(path string, astN *parser.AstNode) (*Resource, error) {
	switch astN.Type {
	case parser.AstResource:
		return NewResource(path, astN)
	case parser.AstTemplate:
		return nil, errors.Errorf("can't handle templates directly, use the include mechanism!")
	default:
		return nil, errors.Errorf("unexpected node seen: %s", astN.Type)
	}
}

func Provision(res *Resource) error {
	l := glog.New()
	if err := res.Prepare(l); err != nil {
		return err
	}

	if err := res.Generate(l); err != nil {
		return err
	}

	return res.Provision(l)
}
