package smutje

import (
	"log"

	"github.com/gfrey/gconn"
	"github.com/gfrey/smutje/parser"
	"github.com/pkg/errors"
)

type smScript interface {
	Prepare(attrs Attributes, prevHash string) (string, error)
	Exec(l *log.Logger, client gconn.Client) error
	Hash() string
	MustExecute() bool
}

func newScript(path string, n *parser.AstNode) (smScript, error) {
	if n.Type != parser.AstScript {
		return nil, errors.Errorf("expected script node, got %s", n.Type)
	}

	switch s := n.Value.(type) {
	case *parser.SmutjeScript:
		return &smutjeScript{Path: path, ID: n.ID, rawCommand: s.Command}, nil
	case *parser.BashScript:
		return &bashScript{ID: n.ID, Script: s.Script}, nil
	default:
		return nil, errors.Errorf("expected a string value, got %T", n.Value)
	}
}
