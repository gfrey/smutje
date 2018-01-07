package parser

import (
	"strings"
	"testing"
)

func TestParser_TitleFailure(t *testing.T) {
	tt := []struct {
		input string
		err   string
	}{
		{"foobar", `stdin:1: unexpected token read: text ("foobar") (expected title)`},
		{"\n\nfoobar", `stdin:3: unexpected token read: text ("foobar") (expected title)`},
		{"> foobar", `stdin:1: unexpected token read: arrow (expected title)`},
		{"  foobar", `stdin:1: unexpected token read: indent (expected title)`},
		{"# foobar", `invalid title format "# foobar"`},
		{"# foobar: some more", `invalid title format "# foobar: some more"`},
		{"# foobar: some more [invalid!]", `invalid title format "# foobar: some more [invalid!]"`},
	}

	for _, tti := range tt {
		_, err := ParseString("stdin", tti.input)
		if err == nil {
			t.Errorf("expected error %q, got none", tti.err)
		} else if !strings.HasSuffix(err.Error(), tti.err) {
			t.Errorf("expected error %q, got %q", tti.err, err)
		}
	}
}

func TestParser_Happy(t *testing.T) {
	node, err := Parse("testdata/test_base.smd")
	if err != nil {
		t.Fatalf("didn't expect an error parsing the testdata, got: %s", err)
	}

	expNodeName := "asterix.example.org"
	if node.Name != expNodeName {
		t.Errorf("expected node to have name %q, got %q", expNodeName, node.Name)
	}

	expNodeID := "asterix"
	if node.ID != expNodeID {
		t.Errorf("expected node to have name %q, got %q", expNodeID, node.ID)
	}

	if len(node.Children) != 6 {
		t.Errorf("expected node to have %d children, got %d", 6, len(node.Children))
	}

	{
		tt := []struct {
			idx   int
			nodes []astNodeType
		}{
			{-1, []astNodeType{AstText, AstAttributes, AstAttributes, AstPackage, AstPackage, AstInclude}},
			{3, []astNodeType{AstText, AstText, AstScript, AstText, AstScript}},
			{4, []astNodeType{AstText, AstScript}},
			{5, []astNodeType{AstAttributes}},
		}
		for _, tti := range tt {
			n := node
			if tti.idx != -1 {
				n = node.Children[tti.idx]
			}

			for i, exp := range tti.nodes {
				if i >= len(n.Children) {
					t.Errorf("node %d has less children, than expected!", tti.idx)
					continue
				}
				if n.Children[i].Type != exp {
					t.Errorf("node %d: expected child %d to have type %s, got %s", tti.idx, i, exp, n.Children[i].Type)
				}
			}
		}
	}
}
