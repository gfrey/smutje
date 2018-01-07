package parser

import (
	"regexp"
	"strings"

	"git.gf-hh.net/gmd"
	"github.com/pkg/errors"
)

func ParseString(name, template string) (*AstNode, error) {
	// generalized AST, that we need to pimp to match our style.
	rAst, err := gmd.ParseString(name, template)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse template %q", name)
	}

	return convertSection(rAst)
}

func Parse(filename string) (*AstNode, error) {
	// generalized AST, that we need to pimp to match our style.
	rAst, err := gmd.Parse(filename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse file")
	}

	return convertSection(rAst)
}

var reTitle = regexp.MustCompile(`^\#+ (\w+):[ ]*(.*)[ ]*\[(\w+)\]$`)

func convertSection(raw *gmd.AstNode) (*AstNode, error) {
	if len(raw.Lines) != 1 {
		return nil, raw.Errorf(0, "section title must be a single line")
	}
	val := raw.Lines[0]

	m := reTitle.FindStringSubmatch(val)
	if m == nil {
		return nil, raw.Errorf(0, "invalid title format: %q", val)
	}

	n := &AstNode{
		Type: mapTyp(m[1]),
		Name: strings.TrimSpace(m[2]), ID: m[3],
	}

	for _, child := range raw.Children {
		switch child.Type {
		case gmd.AstSection:
			c, err := convertSection(child)
			if err != nil {
				return nil, err
			}
			n.Children = append(n.Children, c)
		case gmd.AstCode:
			cs, err := convertCode(child)
			if err != nil {
				return nil, err
			}
			n.Children = append(n.Children, cs...)
		case gmd.AstText:
			c, err := convertText(child)
			if err != nil {
				return nil, err
			}
			n.Children = append(n.Children, c)
		case gmd.AstQuote:
			c, err := convertQuote(child)
			if err != nil {
				return nil, err
			}
			n.Children = append(n.Children, c)
		default:
			return nil, errors.Errorf("Invalid node type: %s", child)
		}
	}

	return n, nil
}

func convertCode(raw *gmd.AstNode) ([]*AstNode, error) {
	nodes := []*AstNode{}

	start := 0
	for i, line := range raw.Lines {
		if line != "" && line[0] == ':' {
			nodes = appendScriptIfAny(nodes, raw.Lines, start, i)
			nodes = append(nodes, &AstNode{
				Type:  AstScript,
				Value: &SmutjeScript{raw.Lines[i]},
			})
			start = i + 1
		}
	}
	return appendScriptIfAny(nodes, raw.Lines, start, len(raw.Lines)), nil
}

func appendScriptIfAny(nodes []*AstNode, lines []string, start, cur int) []*AstNode {
	if cur > start {
		return append(nodes, &AstNode{
			Type:  AstScript,
			Value: &BashScript{strings.Join(lines[start:cur], "\n")},
		})
	}
	return nodes
}

func convertText(raw *gmd.AstNode) (*AstNode, error) {
	return &AstNode{Type: AstText, Value: raw.Lines}, nil
}

var reQuote = regexp.MustCompile(`^(\w+):[ ]*(.*)[ ]*$`)

func convertQuote(raw *gmd.AstNode) (*AstNode, error) {
	attrs := []*Attribute{}
	for i, line := range raw.Lines {
		m := reQuote.FindStringSubmatch(line)
		if m == nil {
			return nil, raw.Errorf(i, "invalid attribute format: %q", line)
		}
		attrs = append(attrs, &Attribute{m[1], m[2]})
	}
	return &AstNode{Type: AstAttributes, Value: attrs}, nil
}
