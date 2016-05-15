package smutje

import (
	"fmt"

	"github.com/gfrey/smutje/connection"
	"github.com/gfrey/smutje/logger"
	"github.com/gfrey/smutje/parser"
)

type smScript interface {
	Prepare(attrs smAttributes, prevHash string) (string, error)
	Exec(l logger.Logger, client connection.Client) error
	Hash() string
}

func newScript(path string, n *parser.AstNode) (smScript, error) {
	if n.Type != parser.AstScript {
		return nil, fmt.Errorf("expected script node, got %s", n.Type)
	}

	switch s := n.Value.(type) {
	case *parser.SmutjeScript:
		return &smutjeScript{Path: path, ID: n.ID, rawCommand: s.Command}, nil
	case *parser.BashScript:
		return &bashScript{ID: n.ID, Script: s.Script}, nil
	default:
		return nil, fmt.Errorf("expected a string value, got %T", n.Value)
	}
}
