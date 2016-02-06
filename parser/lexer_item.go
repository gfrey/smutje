package parser

import (
	"fmt"
	"strings"
)

type itemType int

type item struct {
	typ  itemType
	val  string
	line int
}

const (
	itemError itemType = iota
	itemHash
	itemArrow
	itemIndent
	itemText
	itemEOF
	itemEmptyLine
)

func (i itemType) String() string {
	switch i {
	case itemError:
		return "error"
	case itemArrow:
		return "attribute"
	case itemHash:
		return "title"
	case itemEmptyLine:
		return "empty line"
	case itemIndent:
		return "script"
	case itemText:
		return "text"
	case itemEOF:
		return "eof"
	default:
		return fmt.Sprintf("unknown type: %d", i)
	}
}

func (i item) String() string {
	switch i.typ {
	case itemError:
		return fmt.Sprintf("error (%s)", strings.TrimSpace(i.val))
	case itemArrow:
		return "arrow"
	case itemHash:
		return fmt.Sprintf("hash (%d)", len(i.val))
	case itemEmptyLine:
		return "empty line"
	case itemIndent:
		return "indent"
	case itemText:
		return fmt.Sprintf("text (%q)", strings.TrimSpace(i.val))
	case itemEOF:
		return "eof"
	default:
		return fmt.Sprintf("unknown type: %d", i.typ)
	}
}
