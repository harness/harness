package goquery

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// Test helper functions and members
var doc *Document
var doc2 *Document
var doc3 *Document
var docB *Document
var docW *Document

func Doc() *Document {
	if doc == nil {
		doc = loadDoc("page.html")
	}
	return doc
}
func DocClone() *Document {
	return CloneDocument(Doc())
}
func Doc2() *Document {
	if doc2 == nil {
		doc2 = loadDoc("page2.html")
	}
	return doc2
}
func Doc2Clone() *Document {
	return CloneDocument(Doc2())
}
func Doc3() *Document {
	if doc3 == nil {
		doc3 = loadDoc("page3.html")
	}
	return doc3
}
func Doc3Clone() *Document {
	return CloneDocument(Doc3())
}
func DocB() *Document {
	if docB == nil {
		docB = loadDoc("gotesting.html")
	}
	return docB
}
func DocBClone() *Document {
	return CloneDocument(DocB())
}
func DocW() *Document {
	if docW == nil {
		docW = loadDoc("gowiki.html")
	}
	return docW
}
func DocWClone() *Document {
	return CloneDocument(DocW())
}

func assertLength(t *testing.T, nodes []*html.Node, length int) {
	if len(nodes) != length {
		t.Errorf("Expected %d nodes, found %d.", length, len(nodes))
		for i, n := range nodes {
			t.Logf("Node %d: %+v.", i, n)
		}
	}
}

func assertClass(t *testing.T, sel *Selection, class string) {
	if !sel.HasClass(class) {
		t.Errorf("Expected node to have class %s, found %+v.", class, sel.Get(0))
	}
}

func assertPanic(t *testing.T) {
	if e := recover(); e == nil {
		t.Error("Expected a panic.")
	}
}

func assertEqual(t *testing.T, s1 *Selection, s2 *Selection) {
	if s1 != s2 {
		t.Error("Expected selection objects to be the same.")
	}
}

func assertSelectionIs(t *testing.T, sel *Selection, is ...string) {
	for i := 0; i < sel.Length(); i++ {
		if !sel.Eq(i).Is(is[i]) {
			t.Errorf("Expected node %d to be %s, found %+v", i, is[i], sel.Get(i))
		}
	}
}

func printSel(t *testing.T, sel *Selection) {
	if testing.Verbose() {
		h, err := sel.Html()
		if err != nil {
			t.Fatal(err)
		}
		t.Log(h)
	}
}

func loadDoc(page string) *Document {
	var f *os.File
	var e error

	if f, e = os.Open(fmt.Sprintf("./testdata/%s", page)); e != nil {
		panic(e.Error())
	}
	defer f.Close()

	var node *html.Node
	if node, e = html.Parse(f); e != nil {
		panic(e.Error())
	}
	return NewDocumentFromNode(node)
}

func TestNewDocument(t *testing.T) {
	if f, e := os.Open("./testdata/page.html"); e != nil {
		t.Error(e.Error())
	} else {
		defer f.Close()
		if node, e := html.Parse(f); e != nil {
			t.Error(e.Error())
		} else {
			doc = NewDocumentFromNode(node)
		}
	}
}

func TestNewDocumentFromReader(t *testing.T) {
	cases := []struct {
		src string
		err bool
		sel string
		cnt int
	}{
		0: {
			src: `
<html>
<head>
<title>Test</title>
<body>
<h1>Hi</h1>
</body>
</html>`,
			sel: "h1",
			cnt: 1,
		},
		1: {
			// Actually pretty hard to make html.Parse return an error
			// based on content...
			src: `<html><body><aef<eqf>>>qq></body></ht>`,
		},
	}
	buf := bytes.NewBuffer(nil)

	for i, c := range cases {
		buf.Reset()
		buf.WriteString(c.src)

		d, e := NewDocumentFromReader(buf)
		if (e != nil) != c.err {
			if c.err {
				t.Errorf("[%d] - expected error, got none", i)
			} else {
				t.Errorf("[%d] - expected no error, got %s", i, e)
			}
		}
		if c.sel != "" {
			s := d.Find(c.sel)
			if s.Length() != c.cnt {
				t.Errorf("[%d] - expected %d nodes, found %d", i, c.cnt, s.Length())
			}
		}
	}
}

func TestNewDocumentFromResponseNil(t *testing.T) {
	_, e := NewDocumentFromResponse(nil)
	if e == nil {
		t.Error("Expected error, got none")
	}
}

func TestIssue103(t *testing.T) {
	d, err := NewDocumentFromReader(strings.NewReader("<html><title>Scientists Stored These Images in DNA—Then Flawlessly Retrieved Them</title></html>"))
	if err != nil {
		t.Error(err)
	}
	text := d.Find("title").Text()
	for i, r := range text {
		t.Logf("%d: %d - %q\n", i, r, string(r))
	}
	t.Log(text)
}
