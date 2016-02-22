package smutje

import (
	"fmt"

	"github.com/gfrey/smutje/parser"
)

func newBlueprint(n *parser.AstNode) (string, error) {
	if n.Type != parser.AstBlueprint {
		return "", fmt.Errorf("expected script node, got %s", n.Type)
	}

	var bprint string

	for _, child := range n.Children {
		switch child.Type {
		case parser.AstScript:
			if bprint != "" {
				return "", fmt.Errorf("only one blueprint node allowed!")
			}
			bscript, ok := child.Value.(*parser.BashScript)
			if !ok {
				return "", fmt.Errorf("expected a string value, got %T", child.Value)
			}
			bprint = bscript.Script
		case parser.AstText:
			// ignore
		default:
			return "", fmt.Errorf("unexpected node seen: %s", child.Type)
		}
	}

	return bprint, nil
}
