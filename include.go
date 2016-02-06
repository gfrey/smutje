package smutje

import (
	"fmt"

	"os"

	"github.com/gfrey/smutje/parser"
)

func newInclude(parentID string, n *parser.AstNode) ([]*smPackage, error) {
	if n.Type != parser.AstInclude {
		return nil, fmt.Errorf("expected include node, got %s", n.Type)
	}

	switch _, err := os.Lstat(n.Name); {
	case os.IsNotExist(err):
		return nil, fmt.Errorf("template %s does not exist!", n.Name)
	case err != nil:
		return nil, err
	}

	attrs := smAttributes{}
	for _, child := range n.Children {
		switch child.Type {
		case parser.AstAttributes:
			a, err := newAttributes(child)
			if err != nil {
				return nil, err
			}
			attrs = attrs.Merge(a)
		case parser.AstText:
			// ignore
		default:
			return nil, fmt.Errorf("unexpected node seen: %s", child.Type)
		}
	}

	nodeID := n.ID
	if parentID != "" {
		nodeID = parentID + "." + n.ID
	}

	return parseTemplate(n.Name, nodeID, attrs)
}

func parseTemplate(filename, parentID string, attrs smAttributes) ([]*smPackage, error) {
	n, err := parser.Parse(filename)
	if err != nil {
		return nil, err
	}

	if n.Type != parser.AstTemplate {
		return nil, fmt.Errorf("expected template node, got %s", n.Type)
	}

	pkgs := []*smPackage{}

	for _, child := range n.Children {
		switch child.Type {
		case parser.AstAttributes:
			newAttrs, err := newAttributes(child)
			if err != nil {
				return nil, err
			}
			attrs.MergeInplace(newAttrs)
		default:
			npkgs, nattrs, err := handleChild(parentID, child)
			if err != nil {
				return nil, err
			}

			for _, pkg := range npkgs {
				pkg.Attributes = attrs.Merge(nattrs).Merge(pkg.Attributes)
				pkgs = append(pkgs, pkg)
			}
		}
	}

	return pkgs, nil
}
