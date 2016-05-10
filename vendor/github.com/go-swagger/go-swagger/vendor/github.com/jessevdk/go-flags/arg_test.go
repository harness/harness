package flags

import (
	"testing"
)

func TestPositional(t *testing.T) {
	var opts = struct {
		Value bool `short:"v"`

		Positional struct {
			Command  int
			Filename string
			Rest     []string
		} `positional-args:"yes" required:"yes"`
	}{}

	p := NewParser(&opts, Default)
	ret, err := p.ParseArgs([]string{"10", "arg_test.go", "a", "b"})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
		return
	}

	if opts.Positional.Command != 10 {
		t.Fatalf("Expected opts.Positional.Command to be 10, but got %v", opts.Positional.Command)
	}

	if opts.Positional.Filename != "arg_test.go" {
		t.Fatalf("Expected opts.Positional.Filename to be \"arg_test.go\", but got %v", opts.Positional.Filename)
	}

	assertStringArray(t, opts.Positional.Rest, []string{"a", "b"})
	assertStringArray(t, ret, []string{})
}

func TestPositionalRequired(t *testing.T) {
	var opts = struct {
		Value bool `short:"v"`

		Positional struct {
			Command  int
			Filename string
			Rest     []string
		} `positional-args:"yes" required:"yes"`
	}{}

	p := NewParser(&opts, None)
	_, err := p.ParseArgs([]string{"10"})

	assertError(t, err, ErrRequired, "the required argument `Filename` was not provided")
}

func TestPositionalRequiredRest1Fail(t *testing.T) {
	var opts = struct {
		Value bool `short:"v"`

		Positional struct {
			Rest []string `required:"yes"`
		} `positional-args:"yes"`
	}{}

	p := NewParser(&opts, None)
	_, err := p.ParseArgs([]string{})

	assertError(t, err, ErrRequired, "the required argument `Rest (at least 1 argument)` was not provided")
}

func TestPositionalRequiredRest1Pass(t *testing.T) {
	var opts = struct {
		Value bool `short:"v"`

		Positional struct {
			Rest []string `required:"yes"`
		} `positional-args:"yes"`
	}{}

	p := NewParser(&opts, None)
	_, err := p.ParseArgs([]string{"rest1"})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
		return
	}

	if len(opts.Positional.Rest) != 1 {
		t.Fatalf("Expected 1 positional rest argument")
	}

	assertString(t, opts.Positional.Rest[0], "rest1")
}

func TestPositionalRequiredRest2Fail(t *testing.T) {
	var opts = struct {
		Value bool `short:"v"`

		Positional struct {
			Rest []string `required:"2"`
		} `positional-args:"yes"`
	}{}

	p := NewParser(&opts, None)
	_, err := p.ParseArgs([]string{"rest1"})

	assertError(t, err, ErrRequired, "the required argument `Rest (at least 2 arguments, but got only 1)` was not provided")
}

func TestPositionalRequiredRest2Pass(t *testing.T) {
	var opts = struct {
		Value bool `short:"v"`

		Positional struct {
			Rest []string `required:"2"`
		} `positional-args:"yes"`
	}{}

	p := NewParser(&opts, None)
	_, err := p.ParseArgs([]string{"rest1", "rest2", "rest3"})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
		return
	}

	if len(opts.Positional.Rest) != 3 {
		t.Fatalf("Expected 3 positional rest argument")
	}

	assertString(t, opts.Positional.Rest[0], "rest1")
	assertString(t, opts.Positional.Rest[1], "rest2")
	assertString(t, opts.Positional.Rest[2], "rest3")
}
