package goquery

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

var allNodes = `<!doctype html>
<html>
	<head>
		<meta a="b">
	</head>
	<body>
		<p><!-- this is a comment -->
		This is some text.
		</p>
		<div></div>
		<h1 class="header"></h1>
		<h2 class="header"></h2>
	</body>
</html>`

func TestNodeName(t *testing.T) {
	doc, err := NewDocumentFromReader(strings.NewReader(allNodes))
	if err != nil {
		t.Fatal(err)
	}

	n0 := doc.Nodes[0]
	nDT := n0.FirstChild
	sMeta := doc.Find("meta")
	nMeta := sMeta.Get(0)
	sP := doc.Find("p")
	nP := sP.Get(0)
	nComment := nP.FirstChild
	nText := nComment.NextSibling

	cases := []struct {
		node *html.Node
		typ  html.NodeType
		want string
	}{
		{n0, html.DocumentNode, nodeNames[html.DocumentNode]},
		{nDT, html.DoctypeNode, "html"},
		{nMeta, html.ElementNode, "meta"},
		{nP, html.ElementNode, "p"},
		{nComment, html.CommentNode, nodeNames[html.CommentNode]},
		{nText, html.TextNode, nodeNames[html.TextNode]},
	}
	for i, c := range cases {
		got := NodeName(newSingleSelection(c.node, doc))
		if c.node.Type != c.typ {
			t.Errorf("%d: want type %v, got %v", i, c.typ, c.node.Type)
		}
		if got != c.want {
			t.Errorf("%d: want %q, got %q", i, c.want, got)
		}
	}
}

func TestNodeNameMultiSel(t *testing.T) {
	doc, err := NewDocumentFromReader(strings.NewReader(allNodes))
	if err != nil {
		t.Fatal(err)
	}

	in := []string{"p", "h1", "div"}
	var out []string
	doc.Find(strings.Join(in, ", ")).Each(func(i int, s *Selection) {
		got := NodeName(s)
		out = append(out, got)
	})
	sort.Strings(in)
	sort.Strings(out)
	if !reflect.DeepEqual(in, out) {
		t.Errorf("want %v, got %v", in, out)
	}
}

func TestOuterHtml(t *testing.T) {
	doc, err := NewDocumentFromReader(strings.NewReader(allNodes))
	if err != nil {
		t.Fatal(err)
	}

	n0 := doc.Nodes[0]
	nDT := n0.FirstChild
	sMeta := doc.Find("meta")
	sP := doc.Find("p")
	nP := sP.Get(0)
	nComment := nP.FirstChild
	nText := nComment.NextSibling
	sHeaders := doc.Find(".header")

	cases := []struct {
		node *html.Node
		sel  *Selection
		want string
	}{
		{nDT, nil, "<!DOCTYPE html>"}, // render makes DOCTYPE all caps
		{nil, sMeta, `<meta a="b"/>`}, // and auto-closes the meta
		{nil, sP, `<p><!-- this is a comment -->
		This is some text.
		</p>`},
		{nComment, nil, "<!-- this is a comment -->"},
		{nText, nil, `
		This is some text.
		`},
		{nil, sHeaders, `<h1 class="header"></h1>`},
	}
	for i, c := range cases {
		if c.sel == nil {
			c.sel = newSingleSelection(c.node, doc)
		}
		got, err := OuterHtml(c.sel)
		if err != nil {
			t.Fatal(err)
		}

		if got != c.want {
			t.Errorf("%d: want %q, got %q", i, c.want, got)
		}
	}
}
