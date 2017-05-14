package parser

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

func Parse(filename string) (*AstNode, error) {
	input, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read file")
	}

	return ParseString(filename, string(input))
}

func ParseString(name, template string) (*AstNode, error) {
	return parse(name, lex(template))
}

type parser struct {
	input    <-chan item
	queue    []item
	pos      int
	filename string
}

func parse(filename string, in <-chan item) (*AstNode, error) {
	p := &parser{input: in, filename: filename}
	return parseSection(p, nil)
}

func (p *parser) next() item {
	var i item
	if p.pos == len(p.queue) {
		i = <-p.input
		p.queue = append(p.queue, i)
	} else {
		i = p.queue[p.pos]
	}
	p.pos++
	return i
}

func (p *parser) peek() item {
	i := p.next()
	p.backup()
	return i
}

func (p *parser) backup() {
	p.pos--
}

func (p *parser) backupN(n int) {
	p.pos -= n
}

func (p *parser) syntaxErrorf(i item, msg string, args ...interface{}) error {
	prefix := fmt.Sprintf("%s:%d: ", p.filename, i.line)
	return errors.Errorf(prefix+msg, args...)
}

var reTitle = regexp.MustCompile(`^(\w+):[ ]*(.*)[ ]*\[(\w+)\]$`)

const unexpTokenErr = "unexpected token read: %s (expected %s)"

func parseTitle(p *parser) (int, *AstNode, error) {
	switch i := p.peek(); i.typ {
	case itemHash:
		depth := len(i.val)
		p.next()
		i = p.peek()
		if i.typ == itemText {
			val := strings.TrimSpace(p.next().val)

			m := reTitle.FindStringSubmatch(val)
			if m == nil {
				return -1, nil, p.syntaxErrorf(i, "expected title format %q, got: %q", "<Type>: <Name of Section> [<Id>]", val)
			}

			typ := mapTyp(m[1])
			if typ == -1 {
				return -1, nil, p.syntaxErrorf(i, "unexpected section type: %s", m[1])
			}

			return depth, &AstNode{Type: typ, Name: strings.TrimSpace(m[2]), ID: m[3]}, nil
		}
		return -1, nil, p.syntaxErrorf(i, unexpTokenErr, i, itemText)
	default:
		return -1, nil, p.syntaxErrorf(i, unexpTokenErr, i, itemHash)
	}
}

func parseSection(p *parser, parent *AstNode) (*AstNode, error) {
	var node *AstNode
	for {
		switch i := p.peek(); i.typ {
		case itemText:
			if node == nil {
				return nil, p.syntaxErrorf(i, unexpTokenErr, i, itemHash)
			}
			text := parseText(p)
			node.addChild(text)
		case itemIndent:
			if node == nil {
				return nil, p.syntaxErrorf(i, unexpTokenErr, i, itemHash)
			}
			script, err := parseScript(p)
			if err != nil {
				return nil, err
			}
			node.addChild(script)
		case itemArrow:
			if node == nil {
				return nil, p.syntaxErrorf(i, unexpTokenErr, i, itemHash)
			}
			attrs, err := parseAttributes(p)
			if err != nil {
				return nil, err
			}
			node.addChild(attrs)
		case itemHash:
			if node == nil {
				depth, sectionNode, err := parseTitle(p)
				switch {
				case err != nil:
					return nil, err
				case depth > 2, depth == 1 && parent != nil, depth == 2 && parent == nil:
					expDepth := 1
					if parent != nil {
						expDepth = 2
					}
					return nil, p.syntaxErrorf(i, "invalid section depth %d (expected %d)", depth, expDepth)
				default:
					node = sectionNode
				}
			} else if parent != nil {
				return node, nil
			} else {
				sectionNode, err := parseSection(p, node)
				if err != nil {
					return nil, err
				}
				node.addChild(sectionNode)
			}
		case itemEmptyLine:
			p.next() // ignore
		case itemEOF:
			return node, nil
		}
	}
}

func parseAttributes(p *parser) (*AstNode, error) {
	attrs := []*Attribute{}
	for {
		switch i := p.peek(); i.typ {
		case itemArrow:
			p.next()
			if i := p.peek(); i.typ == itemText {
				val := p.next().val
				parts := strings.SplitN(val, ":", 2)
				if len(parts) != 2 {
					return nil, p.syntaxErrorf(i, "attribute must have format %q, got: %q", "key: value", val)
				}
				attrs = append(attrs, &Attribute{strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])})
			} else {
				return nil, p.syntaxErrorf(i, unexpTokenErr, i, itemArrow)
			}
		default:
			return &AstNode{Type: AstAttributes, Value: attrs}, nil
		}
	}
}

func parseText(p *parser) *AstNode {
	desc := []string{}
	for {
		switch i := p.peek(); i.typ {
		case itemText:
			desc = append(desc, strings.TrimSpace(i.val))
			p.next()
		default:
			return &AstNode{Type: AstText, Value: strings.Join(desc, "\n") + "\n"}
		}
	}
}

func parseScript(p *parser) (*AstNode, error) {
	indent := ""
	lines := []string{}
	for {
		switch i := p.peek(); i.typ {
		case itemIndent:
			ind := p.next()
			if indent == "" {
				indent = ind.val
			}
			prefix := strings.TrimPrefix(ind.val, indent)

			txt := p.next()
			if txt.typ != itemText {
				return nil, p.syntaxErrorf(txt, unexpTokenErr, txt, itemIndent)
			}

			val := strings.TrimSpace(txt.val)
			if val == "" || val[0] != ':' {
				lines = append(lines, prefix+val)
				continue
			} else if len(lines) == 0 {
				return &AstNode{Type: AstScript, Value: &SmutjeScript{val}}, nil
			}

			p.backupN(2)
			fallthrough
		default:
			return &AstNode{Type: AstScript, Value: &BashScript{strings.Join(lines, "\n")}}, nil
		}
	}
}
