package smutje

import (
	"fmt"
	"path/filepath"

	"os"

	"github.com/gfrey/smutje/parser"
)

func newInclude(parentID, path string, attrs smAttributes, n *parser.AstNode) ([]*smPackage, error) {
	if n.Type != parser.AstInclude {
		return nil, fmt.Errorf("expected include node, got %s", n.Type)
	}

	filename := filepath.Join(path, n.Name)
	switch _, err := os.Lstat(filename); {
	case os.IsNotExist(err):
		return nil, fmt.Errorf("template %s does not exist!", n.Name)
	case err != nil:
		return nil, err
	}

	incAttrs := attrs.Copy()
	for _, child := range n.Children {
		switch child.Type {
		case parser.AstAttributes:
			a, err := newAttributes(child)
			if err != nil {
				return nil, err
			}
			if err := incAttrs.MergeInplace(a); err != nil {
				return nil, err
			}
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

	return parseTemplate(filename, nodeID, incAttrs)
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
	tmplAttrs := attrs.Copy()
	for _, child := range n.Children {
		switch child.Type {
		case parser.AstAttributes:
			newAttrs, err := newAttributes(child)
			if err != nil {
				return nil, err
			}
			tmplAttrs.MergeInplace(newAttrs)
		default:
			npkgs, err := handleChild(parentID, filepath.Dir(filename), tmplAttrs, child)
			if err != nil {
				return nil, err
			}

			pkgs = append(pkgs, npkgs...)
		}
	}

	return pkgs, nil
}
