package smutje

import (
	"github.com/gfrey/smutje/parser"
	"github.com/pkg/errors"
)

type smAttributes map[string]string

func newAttributes(n *parser.AstNode) (smAttributes, error) {
	if n.Type != parser.AstAttributes {
		return nil, errors.Errorf("expected attributes node, got %s", n.Type)
	}

	attrs := smAttributes{}

	if raw, ok := n.Value.([]*parser.Attribute); !ok {
		return nil, errors.Errorf("expected attributes on node, got %T", n.Value)
	} else {
		for _, a := range raw {
			attrs[a.Key] = a.Val
		}
	}

	return attrs, nil
}

func (a smAttributes) Copy() smAttributes {
	c := smAttributes{}
	for k, v := range a {
		c[k] = v
	}
	return c
}

func (a smAttributes) Merge(b smAttributes) (smAttributes, error) {
	c := smAttributes{}
	for k, v := range a {
		c[k] = v
	}
	return c, c.MergeInplace(b)
}

func (a smAttributes) MergeInplace(b smAttributes) error {
	var err error
	for k, v := range b {
		if _, found := a[k]; found {
			continue
		}
		a[k], err = renderString("key_"+k, v, a)
		if err != nil {
			return err
		}
	}
	return nil
}
