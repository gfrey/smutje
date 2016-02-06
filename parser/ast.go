package parser

import (
	"fmt"
	"strings"
)

type astNodeType int

func (a astNodeType) String() string {
	switch a {
	case AstResource:
		return "Resource"
	case AstTemplate:
		return "Template"
	case AstInclude:
		return "Include"
	case AstBlueprint:
		return "Blueprint"
	case AstPackage:
		return "Package"
	case AstText:
		return "TextNode"
	case AstScript:
		return "ScriptNode"
	case AstAttributes:
		return "AttributesNode"
	default:
		panic("shouldn't be called")
	}
}

const (
	AstResource astNodeType = iota
	AstTemplate
	AstInclude
	AstBlueprint
	AstPackage
	AstText
	AstScript
	AstAttributes
)

type AstNode struct {
	Children []*AstNode

	Name string
	ID   string

	Type  astNodeType
	Value interface{}
}

func (n *AstNode) addChild(child *AstNode) {
	n.Children = append(n.Children, child)
}

func (n *AstNode) String() string {
	switch n.Type {
	case AstResource, AstTemplate:
		content := fmt.Sprintf("# %s: %s [%s]\n\n", n.Type, n.Name, n.ID)
		for i, child := range n.Children {
			if i > 0 {
				content += "\n"
			}
			content += child.String()
		}
		return content
	case AstPackage, AstInclude, AstBlueprint:
		content := fmt.Sprintf("\n## %s: %s [%s]\n\n", n.Type, n.Name, n.ID)
		for i, child := range n.Children {
			if i > 0 {
				content += "\n"
			}
			content += child.String()
		}
		return content
	case AstText:
		return n.Value.(string)
	case AstAttributes:
		content := ""
		for _, attr := range n.Value.([]*Attribute) {
			content += fmt.Sprintf("> %s: %s\n", attr.Key, attr.Val)
		}
		return content
	case AstScript:
		indent := "    "
		switch s := n.Value.(type) {
		case *SmutjeScript:
			return s.IndentedString(indent)
		case *BashScript:
			return s.IndentedString(indent)
		default:
			panic(fmt.Sprintf("unknown script type: %T", n.Value))
		}
	default:
		panic("shouldn't be reached")
	}
}

type Attribute struct {
	Key string
	Val string
}

type BashScript struct {
	Script string
}

func (bs *BashScript) IndentedString(indent string) string {
	lines := []string{}
	for _, line := range strings.Split(bs.Script, "\n") {
		lines = append(lines, indent+line)
	}
	return strings.Join(lines, "\n") + "\n"
}

type SmutjeScript struct {
	Command string
}

func (ss *SmutjeScript) IndentedString(indent string) string {
	return indent + ss.Command + "\n"
}

func mapTyp(typ string) astNodeType {
	switch strings.ToLower(typ) {
	case "blueprint":
		return AstBlueprint
	case "package":
		return AstPackage
	case "resource":
		return AstResource
	case "include":
		return AstInclude
	case "template":
		return AstTemplate
	default:
		return -1
	}
}
