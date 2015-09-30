package raven

import (
	"fmt"
	"go/build"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type FunctionNameTest struct {
	skip int
	pack string
	name string
}

var (
	thisFile          string
	thisPackage       string
	functionNameTests []FunctionNameTest
)

func TestFunctionName(t *testing.T) {
	for _, test := range functionNameTests {
		pc, _, _, _ := runtime.Caller(test.skip)
		pack, name := functionName(pc)

		if pack != test.pack {
			t.Errorf("incorrect package; got %s, want %s", pack, test.pack)
		}
		if name != test.name {
			t.Errorf("incorrect function; got %s, want %s", name, test.name)
		}
	}
}

func TestStacktrace(t *testing.T) {
	st := trace()
	if st == nil {
		t.Error("got nil stacktrace")
	}
	if len(st.Frames) == 0 {
		t.Error("got zero frames")
	}

	f := st.Frames[len(st.Frames)-1]
	if f.Filename != thisFile {
		t.Errorf("incorrect Filename; got %s, want %s", f.Filename, thisFile)
	}
	if !strings.HasSuffix(f.AbsolutePath, thisFile) {
		t.Error("incorrect AbsolutePath:", f.AbsolutePath)
	}
	if f.Function != "trace" {
		t.Error("incorrect Function:", f.Function)
	}
	if f.Module != thisPackage {
		t.Error("incorrect Module:", f.Module)
	}
	if f.Lineno != 83 {
		t.Error("incorrect Lineno:", f.Lineno)
	}
	if f.ContextLine != "\treturn NewStacktrace(0, 2, []string{thisPackage})" {
		t.Errorf("incorrect ContextLine: %#v", f.ContextLine)
	}
	if len(f.PreContext) != 2 || f.PreContext[0] != "// a" || f.PreContext[1] != "func trace() *Stacktrace {" {
		t.Errorf("incorrect PreContext %#v", f.PreContext)
	}
	if len(f.PostContext) != 2 || f.PostContext[0] != "\t// b" || f.PostContext[1] != "}" {
		t.Errorf("incorrect PostContext %#v", f.PostContext)
	}
	if !f.InApp {
		t.Error("expected InApp to be true")
	}

	if st.Culprit() != fmt.Sprintf("%s.trace", thisPackage) {
		t.Error("incorrect Culprit:", st.Culprit())
	}
}

// a
func trace() *Stacktrace {
	return NewStacktrace(0, 2, []string{thisPackage})
	// b
}

func derivePackage() (file, pack string) {
	// Get file name by seeking caller's file name.
	_, callerFile, _, ok := runtime.Caller(1)
	if !ok {
		return
	}

	// Trim file name
	file = callerFile
	for _, dir := range build.Default.SrcDirs() {
		dir := dir + string(filepath.Separator)
		if trimmed := strings.TrimPrefix(callerFile, dir); len(trimmed) < len(file) {
			file = trimmed
		}
	}

	// Now derive package name
	dir := filepath.Dir(callerFile)

	dirPkg, err := build.ImportDir(dir, build.AllowBinary)
	if err != nil {
		return
	}

	pack = dirPkg.ImportPath
	return
}

func init() {
	thisFile, thisPackage = derivePackage()
	functionNameTests = []FunctionNameTest{
		{0, thisPackage, "TestFunctionName"},
		{1, "testing", "tRunner"},
		{2, "runtime", "goexit"},
		{100, "", ""},
	}
}

// TestNewStacktrace_outOfBounds verifies that a context exceeding the number
// of lines in a file does not cause a panic.
func TestNewStacktrace_outOfBounds(t *testing.T) {
	st := NewStacktrace(0, 1000000, []string{thisPackage})
	f := st.Frames[len(st.Frames)-1]
	if f.ContextLine != "\tst := NewStacktrace(0, 1000000, []string{thisPackage})" {
		t.Errorf("incorrect ContextLine: %#v", f.ContextLine)
	}
}
