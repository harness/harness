package main

import (
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

type sourceFile struct {
	Name     string
	Contents []byte
}

func expectBoxes(expected []string, actual map[string]bool) error {
	if len(expected) != len(actual) {
		return fmt.Errorf("expected %v, got %v", expected, actual)
	}
	for _, box := range expected {
		if _, ok := actual[box]; !ok {
			return fmt.Errorf("expected %v, got %v", expected, actual)
		}
	}
	return nil
}

func setUpTestPkg(pkgName string, files []sourceFile) (*build.Package, func(), error) {
	temp, err := ioutil.TempDir("", "go.rice-test")
	if err != nil {
		return nil, func() {}, err
	}
	cleanup := func() {
		os.RemoveAll(temp)
	}
	dir := filepath.Join(temp, pkgName)
	if err := os.Mkdir(dir, 0770); err != nil {
		return nil, cleanup, err
	}
	for _, f := range files {
		if err := ioutil.WriteFile(filepath.Join(dir, f.Name), f.Contents, 0660); err != nil {
			return nil, cleanup, err
		}
	}
	pkg, err := build.ImportDir(dir, 0)
	return pkg, cleanup, err
}

func TestFindOneBox(t *testing.T) {
	pkg, cleanup, err := setUpTestPkg("foobar", []sourceFile{
		{
			"boxes.go",
			[]byte(`package main

import (
	"github.com/GeertJohan/go.rice"
)

func main() {
	rice.MustFindBox("foo")
}
`),
		},
	})
	defer cleanup()
	if err != nil {
		t.Error(err)
		return
	}

	expectedBoxes := []string{"foo"}
	boxMap := findBoxes(pkg)
	if err := expectBoxes(expectedBoxes, boxMap); err != nil {
		t.Error(err)
	}
}

func TestFindMultipleBoxes(t *testing.T) {
	pkg, cleanup, err := setUpTestPkg("foobar", []sourceFile{
		{
			"boxes.go",
			[]byte(`package main

import (
	"github.com/GeertJohan/go.rice"
)

func main() {
	rice.MustFindBox("foo")
	rice.MustFindBox("bar")
}
`),
		},
	})
	defer cleanup()
	if err != nil {
		t.Error(err)
		return
	}

	expectedBoxes := []string{"foo", "bar"}
	boxMap := findBoxes(pkg)
	if err := expectBoxes(expectedBoxes, boxMap); err != nil {
		t.Error(err)
	}
}

func TestNoBoxFoundIfRiceNotImported(t *testing.T) {
	pkg, cleanup, err := setUpTestPkg("foobar", []sourceFile{
		{
			"boxes.go",
			[]byte(`package main
type fakerice struct {}

func (fr fakerice) FindBox(s string) {
}

func main() {
	rice := fakerice{}
	rice.FindBox("foo")
}
`),
		},
	})
	defer cleanup()
	if err != nil {
		t.Error(err)
		return
	}

	boxMap := findBoxes(pkg)
	if _, ok := boxMap["foo"]; ok {
		t.Errorf("Unexpected box %q was found", "foo")
	}
}

func TestUnrelatedBoxesAreNotFound(t *testing.T) {
	pkg, cleanup, err := setUpTestPkg("foobar", []sourceFile{
		{
			"boxes.go",
			[]byte(`package foobar

import (
	_ "github.com/GeertJohan/go.rice"
)

type fakerice struct {}

func (fr fakerice) FindBox(s string) {
}

func FindBox(s string) {

}

func LoadBoxes() {
	rice := fakerice{}
	rice.FindBox("foo")
	
	FindBox("bar")
}
`),
		},
	})
	defer cleanup()
	if err != nil {
		t.Error(err)
		return
	}

	boxMap := findBoxes(pkg)
	for _, box := range []string{"foo", "bar"} {
		if _, ok := boxMap[box]; ok {
			t.Errorf("Unexpected box %q was found", box)
		}
	}
}

func TestMixGoodAndBadBoxes(t *testing.T) {
	pkg, cleanup, err := setUpTestPkg("foobar", []sourceFile{
		{
			"boxes1.go",
			[]byte(`package foobar

import (
	_ "github.com/GeertJohan/go.rice"
)

type fakerice struct {}

func (fr fakerice) FindBox(s string) {
}

func FindBox(s string) {

}

func LoadBoxes1() {
	rice := fakerice{}
	rice.FindBox("foo")
	
	FindBox("bar")
}
`),
		},
		{
			"boxes2.go",
			[]byte(`package foobar

import (
	noodles "github.com/GeertJohan/go.rice"
)

func LoadBoxes2() {
	FindBox("baz")
	noodles.FindBox("veggies")
}
`),
		},
		{
			"boxes3.go",
			[]byte(`package foobar

import (
	"github.com/GeertJohan/go.rice"
)

func LoadBoxes3() {
	rice.FindBox("fish")
}
`),
		},
		{
			"boxes4.go",
			[]byte(`package foobar

import (
	. "github.com/GeertJohan/go.rice"
)

func LoadBoxes3() {
	MustFindBox("chicken")
}
`),
		},
	})
	defer cleanup()
	if err != nil {
		t.Error(err)
		return
	}

	boxMap := findBoxes(pkg)
	for _, box := range []string{"foo", "bar", "baz"} {
		if _, ok := boxMap[box]; ok {
			t.Errorf("Unexpected box %q was found", box)
		}
	}
	for _, box := range []string{"veggies", "fish", "chicken"} {
		if _, ok := boxMap[box]; !ok {
			t.Errorf("Expected box %q not found", box)
		}
	}
}
