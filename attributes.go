package smutje

import (
	"fmt"

	"github.com/gfrey/smutje/parser"
)

type smAttributes map[string]string

func newAttributes(n *parser.AstNode) (smAttributes, error) {
	if n.Type != parser.AstAttributes {
		return nil, fmt.Errorf("expected attributes node, got %s", n.Type)
	}

	attrs := smAttributes{}

	if raw, ok := n.Value.([]*parser.Attribute); !ok {
		return nil, fmt.Errorf("expected attributes on node, got %T", n.Value)
	} else {
		for _, a := range raw {
			attrs[a.Key] = a.Val
		}
	}

	return attrs, nil
}

func (a smAttributes) Merge(b smAttributes) smAttributes {
	c := map[string]string{}
	for _, m := range []map[string]string{a, b} {
		for k, v := range m {
			c[k] = v
		}
	}
	return c
}

func (a smAttributes) MergeInplace(b smAttributes) {
	for _, m := range []map[string]string{a, b} {
		for k, v := range m {
			a[k] = v
		}
	}
}
