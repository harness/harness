package main

import (
	"log"
	"os"
	"strings"
	"text/template"
)

var (
	tmpl     *template.Template
	tmplTest *template.Template
)

var types = []string{
	"int", "int8", "int16", "int32", "int64",
	"uint", "uint8", "uint16", "uint32", "uint64",
}

func init() {
	var err error
	tmpl, err = template.New("tmpl").Parse(`package incremental

import (
	"sync"
)

type {{.Upper}} struct {
	increment {{.Lower}}
	lock      sync.Mutex
}

// Next returns with an integer that is exactly one higher as the previous call to Next() for this {{.Upper}}
func (i *{{.Upper}}) Next() {{.Lower}} {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.increment++
	return i.increment
}

// Last returns the number ({{.Lower}}) that was returned by the most recent call to this instance's Next()
func (i *{{.Upper}}) Last() {{.Lower}} {
	return i.increment
}

// Set changes the increment to given value, the succeeding call to Next() will return the given value+1
func (i *{{.Upper}}) Set(value {{.Lower}}) {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.increment = value
}
`)

	tmplTest, err = template.New("tmplTest").Parse(`package incremental

import (
	"testing"
)

type some{{.Upper}}Struct struct {
	i {{.Upper}}
}

func Test{{.Upper}}Ptr(t *testing.T) {
	i := &{{.Upper}}{}
	num := i.Next()
	if num != 1 {
		t.Fatal("expected 1, got %d", num)
	}
	num = i.Next()
	if num != 2 {
		t.Fatal("expected 2, got %d", num)
	}
	num = i.Last()
	if num != 2 {
		t.Fatal("expected last to be 2, got %d", num)
	}

	i.Set(42)
	num = i.Last()
	if num != 42 {
		t.Fatal("expected last to be 42, got %d", num)
	}
	num = i.Next()
	if num != 43 {
		t.Fatal("expected 43, got %d", num)
	}
}

func Test{{.Upper}}AsField(t *testing.T) {
	s := some{{.Upper}}Struct{}
	num := s.i.Next()
	if num != 1 {
		t.Fatal("expected 1, got %d", num)
	}
	num = s.i.Next()
	if num != 2 {
		t.Fatal("expected 2, got %d", num)
	}
	num = s.i.Last()
	if num != 2 {
		t.Fatal("expected last to be 2, got %d", num)
	}

	useSome{{.Upper}}Struct(&s, t)

	num = s.i.Last()
	if num != 3 {
		t.Fatal("expected last to be 3, got %d", num)
	}

	s.i.Set(42)
	num = s.i.Last()
	if num != 42 {
		t.Fatal("expected last to be 42, got %d", num)
	}
	num = s.i.Next()
	if num != 43 {
		t.Fatal("expected 43, got %d", num)
	}
}

func useSome{{.Upper}}Struct(s *some{{.Upper}}Struct, t *testing.T) {
	num := s.i.Next()
	if num != 3 {
		t.Fatal("expected 3, got %d", num)
	}
}
`)
	if err != nil {
		log.Fatal(err)
	}
}

type data struct {
	Upper string
	Lower string
}

func main() {
	// loop over integer types
	for _, t := range types {
		// create data with upper and lower names for the type
		d := &data{
			Upper: strings.ToUpper(t[0:1]) + t[1:],
			Lower: t,
		}

		// create file for type
		file, err := os.Create(t + ".go")
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		// execute template, write directly to file
		err = tmpl.Execute(file, d)
		if err != nil {
			log.Fatal(err)
		}

		// create file for test
		file, err = os.Create(t + "_test.go")
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		// execute template, write directly to file
		err = tmplTest.Execute(file, d)
		if err != nil {
			log.Fatal(err)
		}
	}

	// create doc.go
	file, err := os.Create("doc.go")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	file.WriteString(`
// package incremental provides concurency-safe incremental numbers.
//
// This package was created by a simple piece of code located in the gen subdirectory. Please modify that command if you want to modify this package.
package incremental`)
}
