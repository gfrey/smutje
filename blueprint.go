package smutje

import (
	"github.com/gfrey/smutje/parser"
	"github.com/pkg/errors"
)

func newBlueprint(n *parser.AstNode) (string, error) {
	if n.Type != parser.AstBlueprint {
		return "", errors.Errorf("expected script node, got %s", n.Type)
	}

	var bprint string

	for _, child := range n.Children {
		switch child.Type {
		case parser.AstScript:
			if bprint != "" {
				return "", errors.Errorf("only one blueprint node allowed!")
			}
			bscript, ok := child.Value.(*parser.BashScript)
			if !ok {
				return "", errors.Errorf("expected a string value, got %T", child.Value)
			}
			bprint = bscript.Script
		case parser.AstText:
			// ignore
		default:
			return "", errors.Errorf("unexpected node seen: %s", child.Type)
		}
	}

	return bprint, nil
}
