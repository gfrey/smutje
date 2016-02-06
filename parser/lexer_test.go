package parser

import (
	"strings"
	"testing"
)

func TestLexer(t *testing.T) {
	tt := []struct {
		input string
		items []item
	}{
		{"# foobar", []item{{itemHash, "#", 0}, {itemText, " foobar", 0}}},
		{"## foobar", []item{{itemHash, "##", 0}, {itemText, " foobar", 0}}},
		{"  \tfoo", []item{{itemIndent, "  \t", 0}, {itemText, "foo", 0}}},
		{"> key=value", []item{{itemArrow, ">", 0}, {itemText, " key=value", 0}}},
		{" # foobar", []item{{itemIndent, " ", 0}, {itemText, "# foobar", 0}}},
	}

	for idx, tti := range tt {
		c := lex(tti.input)

		for _, iExp := range tti.items {
			i, ok := <-c
			if !ok {
				t.Fatalf("%d: expected to read item from channel, but channel was closed", idx)
			}
			if i.typ != iExp.typ {
				t.Errorf("%d: expected item of type %q, got %q", idx, iExp, i)
			}
			if i.val != iExp.val {
				t.Errorf("%d: expected item value %q, got %q", idx, iExp.val, i.val)
			}
		}
	}
}

func TestLexer_LineNumber(t *testing.T) {
	lines := []string{
		"# First Line",
		"Second Line",
		"  Third Line",
		"",
		"> Fourth Line",
	}

	c := lex(strings.Join(lines, "\n"))
	items := []item{}
	for item := range c {
		items = append(items, item)
	}

	tt := []item{
		{itemHash, "#", 1}, {itemText, " First Line\n", 1},
		{itemText, "Second Line\n", 2},
		{itemIndent, "  ", 3}, {itemText, "Third Line\n", 3},
		{itemEmptyLine, "\n", 4},
		{itemArrow, ">", 5}, {itemText, " Fourth Line", 5},
		{itemEOF, "", 6},
	}

	if len(tt) != len(items) {
		t.Fatalf("expected to get %d items, got %d", len(tt), len(items))
	}

	for i, iExp := range tt {
		if items[i].typ != iExp.typ {
			t.Errorf("%d: expected item of type %q, got %q", i, iExp, i)
		}
		if items[i].val != iExp.val {
			t.Errorf("%d: expected item value %q, got %q", i, iExp.val, items[i].val)
		}
		if items[i].line != iExp.line {
			t.Errorf("%d: expected item to have line %d, got %d", i, iExp.line, items[i].line)
		}
	}
}
