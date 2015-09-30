package amber

import (
	"bytes"
	"strings"
	"testing"
)

func Test_Doctype(t *testing.T) {
	res, err := run(`!!! 5`, nil)

	if err != nil {
		t.Fatal(err.Error())
	} else {
		expect(res, `<!DOCTYPE html>`, t)
	}
}

func Test_Nesting(t *testing.T) {
	res, err := run(`html
						head
							title
						body`, nil)

	if err != nil {
		t.Fatal(err.Error())
	} else {
		expect(res, `<html><head><title></title></head><body></body></html>`, t)
	}
}

func Test_Mixin(t *testing.T) {
	res, err := run(`
		mixin a($a)
			p #{$a}

		+a(1)`, nil)

	if err != nil {
		t.Fatal(err.Error())
	} else {
		expect(res, `<p>1</p>`, t)
	}
}

func Test_Mixin_NoArguments(t *testing.T) {
	res, err := run(`
		mixin a()
			p Testing

		+a()`, nil)

	if err != nil {
		t.Fatal(err.Error())
	} else {
		expect(res, `<p>Testing</p>`, t)
	}
}

func Test_Mixin_MultiArguments(t *testing.T) {
	res, err := run(`
		mixin a($a, $b, $c, $d)
			p #{$a} #{$b} #{$c} #{$d}

		+a("a", "b", "c", A)`, map[string]int{"A": 2})

	if err != nil {
		t.Fatal(err.Error())
	} else {
		expect(res, `<p>a b c 2</p>`, t)
	}
}

func Test_ClassName(t *testing.T) {
	res, err := run(`div.test
						p.test1.test2
							[class=$]
							.test3`, "test4")

	if err != nil {
		t.Fatal(err.Error())
	} else {
		expect(res, `<div class="test"><p class="test1 test2 test4 test3"></p></div>`, t)
	}
}

func Test_Id(t *testing.T) {
	res, err := run(`div#test
						p#test1#test2`, nil)

	if err != nil {
		t.Fatal(err.Error())
	} else {
		expect(res, `<div id="test"><p id="test2"></p></div>`, t)
	}
}

func Test_Attribute(t *testing.T) {
	res, err := run(`div[name="Test"][foo="bar"].testclass
						p
							[style="text-align: center; color: maroon"]`, nil)

	if err != nil {
		t.Fatal(err.Error())
	} else {
		expect(res, `<div class="testclass" foo="bar" name="Test"><p style="text-align: center; color: maroon"></p></div>`, t)
	}
}

func Test_EmptyAttribute(t *testing.T) {
	res, err := run(`div[name]`, nil)

	if err != nil {
		t.Fatal(err.Error())
	} else {
		expect(res, `<div name></div>`, t)
	}
}

func Test_RawText(t *testing.T) {
	res, err := run(`html
						script
							var a = 5;
							alert(a)
						style
							body {
								color: white
							}`, nil)

	if err != nil {
		t.Fatal(err.Error())
	} else {
		expect(res, "<html><script>var a = 5;\nalert(a)</script><style>body {\n\tcolor: white\n}</style></html>", t)
	}
}

func Test_Empty(t *testing.T) {
	res, err := run(``, nil)

	if err != nil {
		t.Fatal(err.Error())
	} else {
		expect(res, ``, t)
	}
}

func Test_ArithmeticExpression(t *testing.T) {
	res, err := run(`#{A + B * C}`, map[string]int{"A": 2, "B": 3, "C": 4})

	if err != nil {
		t.Fatal(err.Error())
	} else {
		expect(res, `14`, t)
	}
}

func Test_BooleanExpression(t *testing.T) {
	res, err := run(`#{C - A < B}`, map[string]int{"A": 2, "B": 3, "C": 4})

	if err != nil {
		t.Fatal(err.Error())
	} else {
		expect(res, `true`, t)
	}
}

func Test_FuncCall(t *testing.T) {
	res, err := run(`div[data-map=json($)]`, map[string]int{"A": 2, "B": 3, "C": 4})

	if err != nil {
		t.Fatal(err.Error())
	} else {
		expect(res, `<div data-map="{&#34;A&#34;:2,&#34;B&#34;:3,&#34;C&#34;:4}"></div>`, t)
	}
}

func Test_Multiple_File_Inheritance(t *testing.T) {
	tmpl, err := CompileDir("samples/", DefaultDirOptions, DefaultOptions)
	if err != nil {
		t.Fatal(err.Error())
	}

	t1a, ok := tmpl["multilevel.inheritance.a"]
	if ok != true || t1a == nil {
		t.Fatal("CompileDir, template not found.")
	}

	t1b, ok := tmpl["multilevel.inheritance.b"]
	if ok != true || t1b == nil {
		t.Fatal("CompileDir, template not found.")
	}

	t1c, ok := tmpl["multilevel.inheritance.c"]
	if ok != true || t1c == nil {
		t.Fatal("CompileDir, template not found.")
	}

	var res bytes.Buffer
	t1c.Execute(&res, nil)
	expect(strings.TrimSpace(res.String()), "<p>This is C</p>", t)
}

func Failing_Test_CompileDir(t *testing.T) {
	tmpl, err := CompileDir("samples/", DefaultDirOptions, DefaultOptions)

	// Test Compilation
	if err != nil {
		t.Fatal(err.Error())
	}

	// Make sure files are added to map correctly
	val1, ok := tmpl["basic"]
	if ok != true || val1 == nil {
		t.Fatal("CompileDir, template not found.")
	}
	val2, ok := tmpl["inherit"]
	if ok != true || val2 == nil {
		t.Fatal("CompileDir, template not found.")
	}
	val3, ok := tmpl["compiledir_test/basic"]
	if ok != true || val3 == nil {
		t.Fatal("CompileDir, template not found.")
	}
	val4, ok := tmpl["compiledir_test/compiledir_test/basic"]
	if ok != true || val4 == nil {
		t.Fatal("CompileDir, template not found.")
	}

	// Make sure file parsing is the same
	var doc1, doc2 bytes.Buffer
	val1.Execute(&doc1, nil)
	val4.Execute(&doc2, nil)
	expect(doc1.String(), doc2.String(), t)

	// Check against CompileFile
	compilefile, err := CompileFile("samples/basic.amber", DefaultOptions)
	if err != nil {
		t.Fatal(err.Error())
	}
	var doc3 bytes.Buffer
	compilefile.Execute(&doc3, nil)
	expect(doc1.String(), doc3.String(), t)
	expect(doc2.String(), doc3.String(), t)

}

func Benchmark_Parse(b *testing.B) {
	code := `
	!!! 5
	html
		head
			title Test Title
		body
			nav#mainNav[data-foo="bar"]
			div#content
				div.left
				div.center
					block center
						p Main Content
							.long ? somevar && someothervar
				div.right`

	for i := 0; i < b.N; i++ {
		cmp := New()
		cmp.Parse(code)
	}
}

func Benchmark_Compile(b *testing.B) {
	b.StopTimer()

	code := `
	!!! 5
	html
		head
			title Test Title
		body
			nav#mainNav[data-foo="bar"]
			div#content
				div.left
				div.center
					block center
						p Main Content
							.long ? somevar && someothervar
				div.right`

	cmp := New()
	cmp.Parse(code)

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		cmp.CompileString()
	}
}

func expect(cur, expected string, t *testing.T) {
	if cur != expected {
		t.Fatalf("Expected {%s} got {%s}.", expected, cur)
	}
}

func run(tpl string, data interface{}) (string, error) {
	t, err := Compile(tpl, Options{false, false})
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err = t.Execute(&buf, data); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}
