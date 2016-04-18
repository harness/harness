package flags

import (
	"bytes"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

type TestComplete struct {
}

func (t *TestComplete) Complete(match string) []Completion {
	options := []string{
		"hello world",
		"hello universe",
		"hello multiverse",
	}

	ret := make([]Completion, 0, len(options))

	for _, o := range options {
		if strings.HasPrefix(o, match) {
			ret = append(ret, Completion{
				Item: o,
			})
		}
	}

	return ret
}

var completionTestOptions struct {
	Verbose  bool `short:"v" long:"verbose" description:"Verbose messages"`
	Debug    bool `short:"d" long:"debug" description:"Enable debug"`
	Version  bool `long:"version" description:"Show version"`
	Required bool `long:"required" required:"true" description:"This is required"`

	AddCommand struct {
		Positional struct {
			Filename Filename
		} `positional-args:"yes"`
	} `command:"add" description:"add an item"`

	AddMultiCommand struct {
		Positional struct {
			Filename []Filename
		} `positional-args:"yes"`
	} `command:"add-multi" description:"add multiple items"`

	RemoveCommand struct {
		Other bool     `short:"o"`
		File  Filename `short:"f" long:"filename"`
	} `command:"rm" description:"remove an item"`

	RenameCommand struct {
		Completed TestComplete `short:"c" long:"completed"`
	} `command:"rename" description:"rename an item"`
}

type completionTest struct {
	Args             []string
	Completed        []string
	ShowDescriptions bool
}

var completionTests []completionTest

func init() {
	_, sourcefile, _, _ := runtime.Caller(0)
	completionTestSourcedir := filepath.Join(filepath.SplitList(path.Dir(sourcefile))...)

	completionTestFilename := []string{filepath.Join(completionTestSourcedir, "completion.go"), filepath.Join(completionTestSourcedir, "completion_test.go")}

	completionTests = []completionTest{
		{
			// Short names
			[]string{"-"},
			[]string{"-d", "-v"},
			false,
		},

		{
			// Short names concatenated
			[]string{"-dv"},
			[]string{"-dv"},
			false,
		},

		{
			// Long names
			[]string{"--"},
			[]string{"--debug", "--required", "--verbose", "--version"},
			false,
		},

		{
			// Long names with descriptions
			[]string{"--"},
			[]string{
				"--debug     # Enable debug",
				"--required  # This is required",
				"--verbose   # Verbose messages",
				"--version   # Show version",
			},
			true,
		},

		{
			// Long names partial
			[]string{"--ver"},
			[]string{"--verbose", "--version"},
			false,
		},

		{
			// Commands
			[]string{""},
			[]string{"add", "add-multi", "rename", "rm"},
			false,
		},

		{
			// Commands with descriptions
			[]string{""},
			[]string{
				"add        # add an item",
				"add-multi  # add multiple items",
				"rename     # rename an item",
				"rm         # remove an item",
			},
			true,
		},

		{
			// Commands partial
			[]string{"r"},
			[]string{"rename", "rm"},
			false,
		},

		{
			// Positional filename
			[]string{"add", filepath.Join(completionTestSourcedir, "completion")},
			completionTestFilename,
			false,
		},

		{
			// Multiple positional filename (1 arg)
			[]string{"add-multi", filepath.Join(completionTestSourcedir, "completion")},
			completionTestFilename,
			false,
		},
		{
			// Multiple positional filename (2 args)
			[]string{"add-multi", filepath.Join(completionTestSourcedir, "completion.go"), filepath.Join(completionTestSourcedir, "completion")},
			completionTestFilename,
			false,
		},
		{
			// Multiple positional filename (3 args)
			[]string{"add-multi", filepath.Join(completionTestSourcedir, "completion.go"), filepath.Join(completionTestSourcedir, "completion.go"), filepath.Join(completionTestSourcedir, "completion")},
			completionTestFilename,
			false,
		},

		{
			// Flag filename
			[]string{"rm", "-f", path.Join(completionTestSourcedir, "completion")},
			completionTestFilename,
			false,
		},

		{
			// Flag short concat last filename
			[]string{"rm", "-of", path.Join(completionTestSourcedir, "completion")},
			completionTestFilename,
			false,
		},

		{
			// Flag concat filename
			[]string{"rm", "-f" + path.Join(completionTestSourcedir, "completion")},
			[]string{"-f" + completionTestFilename[0], "-f" + completionTestFilename[1]},
			false,
		},

		{
			// Flag equal concat filename
			[]string{"rm", "-f=" + path.Join(completionTestSourcedir, "completion")},
			[]string{"-f=" + completionTestFilename[0], "-f=" + completionTestFilename[1]},
			false,
		},

		{
			// Flag concat long filename
			[]string{"rm", "--filename=" + path.Join(completionTestSourcedir, "completion")},
			[]string{"--filename=" + completionTestFilename[0], "--filename=" + completionTestFilename[1]},
			false,
		},

		{
			// Flag long filename
			[]string{"rm", "--filename", path.Join(completionTestSourcedir, "completion")},
			completionTestFilename,
			false,
		},

		{
			// Custom completed
			[]string{"rename", "-c", "hello un"},
			[]string{"hello universe"},
			false,
		},
	}
}

func TestCompletion(t *testing.T) {
	p := NewParser(&completionTestOptions, Default)
	c := &completion{parser: p}

	for _, test := range completionTests {
		if test.ShowDescriptions {
			continue
		}

		ret := c.complete(test.Args)
		items := make([]string, len(ret))

		for i, v := range ret {
			items[i] = v.Item
		}

		if !reflect.DeepEqual(items, test.Completed) {
			t.Errorf("Args: %#v, %#v\n  Expected: %#v\n  Got:     %#v", test.Args, test.ShowDescriptions, test.Completed, items)
		}
	}
}

func TestParserCompletion(t *testing.T) {
	for _, test := range completionTests {
		if test.ShowDescriptions {
			os.Setenv("GO_FLAGS_COMPLETION", "verbose")
		} else {
			os.Setenv("GO_FLAGS_COMPLETION", "1")
		}

		tmp := os.Stdout

		r, w, _ := os.Pipe()
		os.Stdout = w

		out := make(chan string)

		go func() {
			var buf bytes.Buffer

			io.Copy(&buf, r)

			out <- buf.String()
		}()

		p := NewParser(&completionTestOptions, None)

		p.CompletionHandler = func(items []Completion) {
			comp := &completion{parser: p}
			comp.print(items, test.ShowDescriptions)
		}

		_, err := p.ParseArgs(test.Args)

		w.Close()

		os.Stdout = tmp

		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}

		got := strings.Split(strings.Trim(<-out, "\n"), "\n")

		if !reflect.DeepEqual(got, test.Completed) {
			t.Errorf("Expected: %#v\nGot: %#v", test.Completed, got)
		}
	}

	os.Setenv("GO_FLAGS_COMPLETION", "")
}
