package smutje

import (
	"github.com/gfrey/smutje/parser"
	"github.com/pkg/errors"
)

type Attributes map[string]string

func newAttributes(n *parser.AstNode) (Attributes, error) {
	if n.Type != parser.AstAttributes {
		return nil, errors.Errorf("expected attributes node, got %s", n.Type)
	}

	attrs := Attributes{}

	if raw, ok := n.Value.([]*parser.Attribute); !ok {
		return nil, errors.Errorf("expected attributes on node, got %T", n.Value)
	} else {
		for _, a := range raw {
			attrs[a.Key] = a.Val
		}
	}

	return attrs, nil
}

func (a Attributes) Copy() Attributes {
	c := Attributes{}
	for k, v := range a {
		c[k] = v
	}
	return c
}

func (a Attributes) Merge(b Attributes) (Attributes, error) {
	c := Attributes{}
	for k, v := range a {
		c[k] = v
	}
	return c, c.MergeInplace(b)
}

func (a Attributes) MergeInplace(b Attributes) error {
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
