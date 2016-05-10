package cascadia

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

type selectorTest struct {
	HTML, selector string
	results        []string
}

func nodeString(n *html.Node) string {
	switch n.Type {
	case html.TextNode:
		return n.Data
	case html.ElementNode:
		return html.Token{
			Type: html.StartTagToken,
			Data: n.Data,
			Attr: n.Attr,
		}.String()
	}
	return ""
}

var selectorTests = []selectorTest{
	{
		`<body><address>This address...</address></body>`,
		"address",
		[]string{
			"<address>",
		},
	},
	{
		`<html><head></head><body></body></html>`,
		"*",
		[]string{
			"",
			"<html>",
			"<head>",
			"<body>",
		},
	},
	{
		`<p id="foo"><p id="bar">`,
		"#foo",
		[]string{
			`<p id="foo">`,
		},
	},
	{
		`<ul><li id="t1"><p id="t1">`,
		"li#t1",
		[]string{
			`<li id="t1">`,
		},
	},
	{
		`<ol><li id="t4"><li id="t44">`,
		"*#t4",
		[]string{
			`<li id="t4">`,
		},
	},
	{
		`<ul><li class="t1"><li class="t2">`,
		".t1",
		[]string{
			`<li class="t1">`,
		},
	},
	{
		`<p class="t1 t2">`,
		"p.t1",
		[]string{
			`<p class="t1 t2">`,
		},
	},
	{
		`<div class="test">`,
		"div.teST",
		[]string{},
	},
	{
		`<p class="t1 t2">`,
		".t1.fail",
		[]string{},
	},
	{
		`<p class="t1 t2">`,
		"p.t1.t2",
		[]string{
			`<p class="t1 t2">`,
		},
	},
	{
		`<p><p title="title">`,
		"p[title]",
		[]string{
			`<p title="title">`,
		},
	},
	{
		`<address><address title="foo"><address title="bar">`,
		`address[title="foo"]`,
		[]string{
			`<address title="foo">`,
		},
	},
	{
		`<p title="tot foo bar">`,
		`[    	title        ~=       foo    ]`,
		[]string{
			`<p title="tot foo bar">`,
		},
	},
	{
		`<p title="hello world">`,
		`[title~="hello world"]`,
		[]string{},
	},
	{
		`<p lang="en"><p lang="en-gb"><p lang="enough"><p lang="fr-en">`,
		`[lang|="en"]`,
		[]string{
			`<p lang="en">`,
			`<p lang="en-gb">`,
		},
	},
	{
		`<p title="foobar"><p title="barfoo">`,
		`[title^="foo"]`,
		[]string{
			`<p title="foobar">`,
		},
	},
	{
		`<p title="foobar"><p title="barfoo">`,
		`[title$="bar"]`,
		[]string{
			`<p title="foobar">`,
		},
	},
	{
		`<p title="foobarufoo">`,
		`[title*="bar"]`,
		[]string{
			`<p title="foobarufoo">`,
		},
	},
	{
		`<p class="t1 t2">`,
		".t1:not(.t2)",
		[]string{},
	},
	{
		`<div class="t3">`,
		`div:not(.t1)`,
		[]string{
			`<div class="t3">`,
		},
	},
	{
		`<ol><li id=1><li id=2><li id=3></ol>`,
		`li:nth-child(odd)`,
		[]string{
			`<li id="1">`,
			`<li id="3">`,
		},
	},
	{
		`<ol><li id=1><li id=2><li id=3></ol>`,
		`li:nth-child(even)`,
		[]string{
			`<li id="2">`,
		},
	},
	{
		`<ol><li id=1><li id=2><li id=3></ol>`,
		`li:nth-child(-n+2)`,
		[]string{
			`<li id="1">`,
			`<li id="2">`,
		},
	},
	{
		`<ol><li id=1><li id=2><li id=3></ol>`,
		`li:nth-child(3n+1)`,
		[]string{
			`<li id="1">`,
		},
	},
	{
		`<ol><li id=1><li id=2><li id=3><li id=4></ol>`,
		`li:nth-last-child(odd)`,
		[]string{
			`<li id="2">`,
			`<li id="4">`,
		},
	},
	{
		`<ol><li id=1><li id=2><li id=3><li id=4></ol>`,
		`li:nth-last-child(even)`,
		[]string{
			`<li id="1">`,
			`<li id="3">`,
		},
	},
	{
		`<ol><li id=1><li id=2><li id=3><li id=4></ol>`,
		`li:nth-last-child(-n+2)`,
		[]string{
			`<li id="3">`,
			`<li id="4">`,
		},
	},
	{
		`<ol><li id=1><li id=2><li id=3><li id=4></ol>`,
		`li:nth-last-child(3n+1)`,
		[]string{
			`<li id="1">`,
			`<li id="4">`,
		},
	},
	{
		`<p>some text <span id="1">and a span</span><span id="2"> and another</span></p>`,
		`span:first-child`,
		[]string{
			`<span id="1">`,
		},
	},
	{
		`<span>a span</span> and some text`,
		`span:last-child`,
		[]string{
			`<span>`,
		},
	},
	{
		`<address></address><p id=1><p id=2>`,
		`p:nth-of-type(2)`,
		[]string{
			`<p id="2">`,
		},
	},
	{
		`<address></address><p id=1><p id=2></p><a>`,
		`p:nth-last-of-type(2)`,
		[]string{
			`<p id="1">`,
		},
	},
	{
		`<address></address><p id=1><p id=2></p><a>`,
		`p:last-of-type`,
		[]string{
			`<p id="2">`,
		},
	},
	{
		`<address></address><p id=1><p id=2></p><a>`,
		`p:first-of-type`,
		[]string{
			`<p id="1">`,
		},
	},
	{
		`<div><p id="1"></p><a></a></div><div><p id="2"></p></div>`,
		`p:only-child`,
		[]string{
			`<p id="2">`,
		},
	},
	{
		`<div><p id="1"></p><a></a></div><div><p id="2"></p><p id="3"></p></div>`,
		`p:only-of-type`,
		[]string{
			`<p id="1">`,
		},
	},
	{
		`<p id="1"><!-- --><p id="2">Hello<p id="3"><span>`,
		`:empty`,
		[]string{
			`<head>`,
			`<p id="1">`,
			`<span>`,
		},
	},
	{
		`<div><p id="1"><table><tr><td><p id="2"></table></div><p id="3">`,
		`div p`,
		[]string{
			`<p id="1">`,
			`<p id="2">`,
		},
	},
	{
		`<div><p id="1"><table><tr><td><p id="2"></table></div><p id="3">`,
		`div table p`,
		[]string{
			`<p id="2">`,
		},
	},
	{
		`<div><p id="1"><div><p id="2"></div><table><tr><td><p id="3"></table></div>`,
		`div > p`,
		[]string{
			`<p id="1">`,
			`<p id="2">`,
		},
	},
	{
		`<p id="1"><p id="2"></p><address></address><p id="3">`,
		`p ~ p`,
		[]string{
			`<p id="2">`,
			`<p id="3">`,
		},
	},
	{
		`<p id="1"></p>
		 <!--comment-->
		 <p id="2"></p><address></address><p id="3">`,
		`p + p`,
		[]string{
			`<p id="2">`,
		},
	},
	{
		`<ul><li></li><li></li></ul><p>`,
		`li, p`,
		[]string{
			"<li>",
			"<li>",
			"<p>",
		},
	},
	{
		`<p id="1"><p id="2"></p><address></address><p id="3">`,
		`p +/*This is a comment*/ p`,
		[]string{
			`<p id="2">`,
		},
	},
	{
		`<p>Text block that <span>wraps inner text</span> and continues</p>`,
		`p:contains("that wraps")`,
		[]string{
			`<p>`,
		},
	},
	{
		`<p>Text block that <span>wraps inner text</span> and continues</p>`,
		`p:containsOwn("that wraps")`,
		[]string{},
	},
	{
		`<p>Text block that <span>wraps inner text</span> and continues</p>`,
		`:containsOwn("inner")`,
		[]string{
			`<span>`,
		},
	},
	{
		`<p>Text block that <span>wraps inner text</span> and continues</p>`,
		`p:containsOwn("block")`,
		[]string{
			`<p>`,
		},
	},
	{
		`<div id="d1"><p id="p1"><span>text content</span></p></div><div id="d2"/>`,
		`div:has(#p1)`,
		[]string{
			`<div id="d1">`,
		},
	},
	{
		`<div id="d1"><p id="p1"><span>contents 1</span></p></div>
		<div id="d2"><p>contents <em>2</em></p></div>`,
		`div:has(:containsOwn("2"))`,
		[]string{
			`<div id="d2">`,
		},
	},
	{
		`<body><div id="d1"><p id="p1"><span>contents 1</span></p></div>
		<div id="d2"><p id="p2">contents <em>2</em></p></div></body>`,
		`body :has(:containsOwn("2"))`,
		[]string{
			`<div id="d2">`,
			`<p id="p2">`,
		},
	},
	{
		`<body><div id="d1"><p id="p1"><span>contents 1</span></p></div>
		<div id="d2"><p id="p2">contents <em>2</em></p></div></body>`,
		`body :haschild(:containsOwn("2"))`,
		[]string{
			`<p id="p2">`,
		},
	},
	{
		`<p id="p1">0123456789</p><p id="p2">abcdef</p><p id="p3">0123ABCD</p>`,
		`p:matches([\d])`,
		[]string{
			`<p id="p1">`,
			`<p id="p3">`,
		},
	},
	{
		`<p id="p1">0123456789</p><p id="p2">abcdef</p><p id="p3">0123ABCD</p>`,
		`p:matches([a-z])`,
		[]string{
			`<p id="p2">`,
		},
	},
	{
		`<p id="p1">0123456789</p><p id="p2">abcdef</p><p id="p3">0123ABCD</p>`,
		`p:matches([a-zA-Z])`,
		[]string{
			`<p id="p2">`,
			`<p id="p3">`,
		},
	},
	{
		`<p id="p1">0123456789</p><p id="p2">abcdef</p><p id="p3">0123ABCD</p>`,
		`p:matches([^\d])`,
		[]string{
			`<p id="p2">`,
			`<p id="p3">`,
		},
	},
	{
		`<p id="p1">0123456789</p><p id="p2">abcdef</p><p id="p3">0123ABCD</p>`,
		`p:matches(^(0|a))`,
		[]string{
			`<p id="p1">`,
			`<p id="p2">`,
			`<p id="p3">`,
		},
	},
	{
		`<p id="p1">0123456789</p><p id="p2">abcdef</p><p id="p3">0123ABCD</p>`,
		`p:matches(^\d+$)`,
		[]string{
			`<p id="p1">`,
		},
	},
	{
		`<p id="p1">0123456789</p><p id="p2">abcdef</p><p id="p3">0123ABCD</p>`,
		`p:not(:matches(^\d+$))`,
		[]string{
			`<p id="p2">`,
			`<p id="p3">`,
		},
	},
	{
		`<div><p id="p1">01234<em>567</em>89</p><div>`,
		`div :matchesOwn(^\d+$)`,
		[]string{
			`<p id="p1">`,
			`<em>`,
		},
	},
	{
		`<ul>
			<li><a id="a1" href="http://www.google.com/finance"/>
			<li><a id="a2" href="http://finance.yahoo.com/"/>
			<li><a id="a2" href="http://finance.untrusted.com/"/>
			<li><a id="a3" href="https://www.google.com/news"/>
			<li><a id="a4" href="http://news.yahoo.com"/>
		</ul>`,
		`[href#=(fina)]:not([href#=(\/\/[^\/]+untrusted)])`,
		[]string{
			`<a id="a1" href="http://www.google.com/finance">`,
			`<a id="a2" href="http://finance.yahoo.com/">`,
		},
	},
	{
		`<ul>
			<li><a id="a1" href="http://www.google.com/finance"/>
			<li><a id="a2" href="http://finance.yahoo.com/"/>
			<li><a id="a3" href="https://www.google.com/news"/>
			<li><a id="a4" href="http://news.yahoo.com"/>
		</ul>`,
		`[href#=(^https:\/\/[^\/]*\/?news)]`,
		[]string{
			`<a id="a3" href="https://www.google.com/news">`,
		},
	},
	{
		`<form>
			<label>Username <input type="text" name="username" /></label>
			<label>Password <input type="password" name="password" /></label>
			<label>Country
				<select name="country">
					<option value="ca">Canada</option>
					<option value="us">United States</option>
				</select>
			</label>
			<label>Bio <textarea name="bio"></textarea></label>
			<button>Sign up</button>
		</form>`,
		`:input`,
		[]string{
			`<input type="text" name="username">`,
			`<input type="password" name="password">`,
			`<select name="country">`,
			`<textarea name="bio">`,
			`<button>`,
		},
	},
}

func TestSelectors(t *testing.T) {
	for _, test := range selectorTests {
		s, err := Compile(test.selector)
		if err != nil {
			t.Errorf("error compiling %q: %s", test.selector, err)
			continue
		}

		doc, err := html.Parse(strings.NewReader(test.HTML))
		if err != nil {
			t.Errorf("error parsing %q: %s", test.HTML, err)
			continue
		}

		matches := s.MatchAll(doc)
		if len(matches) != len(test.results) {
			t.Errorf("wanted %d elements, got %d instead", len(test.results), len(matches))
			continue
		}

		for i, m := range matches {
			got := nodeString(m)
			if got != test.results[i] {
				t.Errorf("wanted %s, got %s instead", test.results[i], got)
			}
		}

		firstMatch := s.MatchFirst(doc)
		if len(test.results) == 0 {
			if firstMatch != nil {
				t.Errorf("MatchFirst: want nil, got %s", nodeString(firstMatch))
			}
		} else {
			got := nodeString(firstMatch)
			if got != test.results[0] {
				t.Errorf("MatchFirst: want %s, got %s", test.results[0], got)
			}
		}
	}
}
