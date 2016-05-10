package flags

import (
	"fmt"
	"testing"
)

func TestCommandInline(t *testing.T) {
	var opts = struct {
		Value bool `short:"v"`

		Command struct {
			G bool `short:"g"`
		} `command:"cmd"`
	}{}

	p, ret := assertParserSuccess(t, &opts, "-v", "cmd", "-g")

	assertStringArray(t, ret, []string{})

	if p.Active == nil {
		t.Errorf("Expected active command")
	}

	if !opts.Value {
		t.Errorf("Expected Value to be true")
	}

	if !opts.Command.G {
		t.Errorf("Expected Command.G to be true")
	}

	if p.Command.Find("cmd") != p.Active {
		t.Errorf("Expected to find command `cmd' to be active")
	}
}

func TestCommandInlineMulti(t *testing.T) {
	var opts = struct {
		Value bool `short:"v"`

		C1 struct {
		} `command:"c1"`

		C2 struct {
			G bool `short:"g"`
		} `command:"c2"`
	}{}

	p, ret := assertParserSuccess(t, &opts, "-v", "c2", "-g")

	assertStringArray(t, ret, []string{})

	if p.Active == nil {
		t.Errorf("Expected active command")
	}

	if !opts.Value {
		t.Errorf("Expected Value to be true")
	}

	if !opts.C2.G {
		t.Errorf("Expected C2.G to be true")
	}

	if p.Command.Find("c1") == nil {
		t.Errorf("Expected to find command `c1'")
	}

	if c2 := p.Command.Find("c2"); c2 == nil {
		t.Errorf("Expected to find command `c2'")
	} else if c2 != p.Active {
		t.Errorf("Expected to find command `c2' to be active")
	}
}

func TestCommandFlagOrder1(t *testing.T) {
	var opts = struct {
		Value bool `short:"v"`

		Command struct {
			G bool `short:"g"`
		} `command:"cmd"`
	}{}

	assertParseFail(t, ErrUnknownFlag, "unknown flag `g'", &opts, "-v", "-g", "cmd")
}

func TestCommandFlagOrder2(t *testing.T) {
	var opts = struct {
		Value bool `short:"v"`

		Command struct {
			G bool `short:"g"`
		} `command:"cmd"`
	}{}

	assertParseSuccess(t, &opts, "cmd", "-v", "-g")

	if !opts.Value {
		t.Errorf("Expected Value to be true")
	}

	if !opts.Command.G {
		t.Errorf("Expected Command.G to be true")
	}
}

func TestCommandFlagOrderSub(t *testing.T) {
	var opts = struct {
		Value bool `short:"v"`

		Command struct {
			G bool `short:"g"`

			SubCommand struct {
				B bool `short:"b"`
			} `command:"sub"`
		} `command:"cmd"`
	}{}

	assertParseSuccess(t, &opts, "cmd", "sub", "-v", "-g", "-b")

	if !opts.Value {
		t.Errorf("Expected Value to be true")
	}

	if !opts.Command.G {
		t.Errorf("Expected Command.G to be true")
	}

	if !opts.Command.SubCommand.B {
		t.Errorf("Expected Command.SubCommand.B to be true")
	}
}

func TestCommandFlagOverride1(t *testing.T) {
	var opts = struct {
		Value bool `short:"v"`

		Command struct {
			Value bool `short:"v"`
		} `command:"cmd"`
	}{}

	assertParseSuccess(t, &opts, "-v", "cmd")

	if !opts.Value {
		t.Errorf("Expected Value to be true")
	}

	if opts.Command.Value {
		t.Errorf("Expected Command.Value to be false")
	}
}

func TestCommandFlagOverride2(t *testing.T) {
	var opts = struct {
		Value bool `short:"v"`

		Command struct {
			Value bool `short:"v"`
		} `command:"cmd"`
	}{}

	assertParseSuccess(t, &opts, "cmd", "-v")

	if opts.Value {
		t.Errorf("Expected Value to be false")
	}

	if !opts.Command.Value {
		t.Errorf("Expected Command.Value to be true")
	}
}

func TestCommandFlagOverrideSub(t *testing.T) {
	var opts = struct {
		Value bool `short:"v"`

		Command struct {
			Value bool `short:"v"`

			SubCommand struct {
				Value bool `short:"v"`
			} `command:"sub"`
		} `command:"cmd"`
	}{}

	assertParseSuccess(t, &opts, "cmd", "sub", "-v")

	if opts.Value {
		t.Errorf("Expected Value to be false")
	}

	if opts.Command.Value {
		t.Errorf("Expected Command.Value to be false")
	}

	if !opts.Command.SubCommand.Value {
		t.Errorf("Expected Command.Value to be true")
	}
}

func TestCommandFlagOverrideSub2(t *testing.T) {
	var opts = struct {
		Value bool `short:"v"`

		Command struct {
			Value bool `short:"v"`

			SubCommand struct {
				G bool `short:"g"`
			} `command:"sub"`
		} `command:"cmd"`
	}{}

	assertParseSuccess(t, &opts, "cmd", "sub", "-v")

	if opts.Value {
		t.Errorf("Expected Value to be false")
	}

	if !opts.Command.Value {
		t.Errorf("Expected Command.Value to be true")
	}
}

func TestCommandEstimate(t *testing.T) {
	var opts = struct {
		Value bool `short:"v"`

		Cmd1 struct {
		} `command:"remove"`

		Cmd2 struct {
		} `command:"add"`
	}{}

	p := NewParser(&opts, None)
	_, err := p.ParseArgs([]string{})

	assertError(t, err, ErrCommandRequired, "Please specify one command of: add or remove")
}

func TestCommandEstimate2(t *testing.T) {
	var opts = struct {
		Value bool `short:"v"`

		Cmd1 struct {
		} `command:"remove"`

		Cmd2 struct {
		} `command:"add"`
	}{}

	p := NewParser(&opts, None)
	_, err := p.ParseArgs([]string{"rmive"})

	assertError(t, err, ErrUnknownCommand, "Unknown command `rmive', did you mean `remove'?")
}

type testCommand struct {
	G        bool `short:"g"`
	Executed bool
	EArgs    []string
}

func (c *testCommand) Execute(args []string) error {
	c.Executed = true
	c.EArgs = args

	return nil
}

func TestCommandExecute(t *testing.T) {
	var opts = struct {
		Value bool `short:"v"`

		Command testCommand `command:"cmd"`
	}{}

	assertParseSuccess(t, &opts, "-v", "cmd", "-g", "a", "b")

	if !opts.Value {
		t.Errorf("Expected Value to be true")
	}

	if !opts.Command.Executed {
		t.Errorf("Did not execute command")
	}

	if !opts.Command.G {
		t.Errorf("Expected Command.C to be true")
	}

	assertStringArray(t, opts.Command.EArgs, []string{"a", "b"})
}

func TestCommandClosest(t *testing.T) {
	var opts = struct {
		Value bool `short:"v"`

		Cmd1 struct {
		} `command:"remove"`

		Cmd2 struct {
		} `command:"add"`
	}{}

	args := assertParseFail(t, ErrUnknownCommand, "Unknown command `addd', did you mean `add'?", &opts, "-v", "addd")

	assertStringArray(t, args, []string{"addd"})
}

func TestCommandAdd(t *testing.T) {
	var opts = struct {
		Value bool `short:"v"`
	}{}

	var cmd = struct {
		G bool `short:"g"`
	}{}

	p := NewParser(&opts, Default)
	c, err := p.AddCommand("cmd", "", "", &cmd)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
		return
	}

	ret, err := p.ParseArgs([]string{"-v", "cmd", "-g", "rest"})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
		return
	}

	assertStringArray(t, ret, []string{"rest"})

	if !opts.Value {
		t.Errorf("Expected Value to be true")
	}

	if !cmd.G {
		t.Errorf("Expected Command.G to be true")
	}

	if p.Command.Find("cmd") != c {
		t.Errorf("Expected to find command `cmd'")
	}

	if p.Commands()[0] != c {
		t.Errorf("Expected command %#v, but got %#v", c, p.Commands()[0])
	}

	if c.Options()[0].ShortName != 'g' {
		t.Errorf("Expected short name `g' but got %v", c.Options()[0].ShortName)
	}
}

func TestCommandNestedInline(t *testing.T) {
	var opts = struct {
		Value bool `short:"v"`

		Command struct {
			G bool `short:"g"`

			Nested struct {
				N string `long:"n"`
			} `command:"nested"`
		} `command:"cmd"`
	}{}

	p, ret := assertParserSuccess(t, &opts, "-v", "cmd", "-g", "nested", "--n", "n", "rest")

	assertStringArray(t, ret, []string{"rest"})

	if !opts.Value {
		t.Errorf("Expected Value to be true")
	}

	if !opts.Command.G {
		t.Errorf("Expected Command.G to be true")
	}

	assertString(t, opts.Command.Nested.N, "n")

	if c := p.Command.Find("cmd"); c == nil {
		t.Errorf("Expected to find command `cmd'")
	} else {
		if c != p.Active {
			t.Errorf("Expected `cmd' to be the active parser command")
		}

		if nested := c.Find("nested"); nested == nil {
			t.Errorf("Expected to find command `nested'")
		} else if nested != c.Active {
			t.Errorf("Expected to find command `nested' to be the active `cmd' command")
		}
	}
}

func TestRequiredOnCommand(t *testing.T) {
	var opts = struct {
		Value bool `short:"v" required:"true"`

		Command struct {
			G bool `short:"g"`
		} `command:"cmd"`
	}{}

	assertParseFail(t, ErrRequired, fmt.Sprintf("the required flag `%cv' was not specified", defaultShortOptDelimiter), &opts, "cmd")
}

func TestRequiredAllOnCommand(t *testing.T) {
	var opts = struct {
		Value   bool `short:"v" required:"true"`
		Missing bool `long:"missing" required:"true"`

		Command struct {
			G bool `short:"g"`
		} `command:"cmd"`
	}{}

	assertParseFail(t, ErrRequired, fmt.Sprintf("the required flags `%smissing' and `%cv' were not specified", defaultLongOptDelimiter, defaultShortOptDelimiter), &opts, "cmd")
}

func TestDefaultOnCommand(t *testing.T) {
	var opts = struct {
		Command struct {
			G string `short:"g" default:"value"`
		} `command:"cmd"`
	}{}

	assertParseSuccess(t, &opts, "cmd")

	if opts.Command.G != "value" {
		t.Errorf("Expected G to be \"value\"")
	}
}

func TestSubcommandsOptional(t *testing.T) {
	var opts = struct {
		Value bool `short:"v"`

		Cmd1 struct {
		} `command:"remove"`

		Cmd2 struct {
		} `command:"add"`
	}{}

	p := NewParser(&opts, None)
	p.SubcommandsOptional = true

	_, err := p.ParseArgs([]string{"-v"})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
		return
	}

	if !opts.Value {
		t.Errorf("Expected Value to be true")
	}
}

func TestCommandAlias(t *testing.T) {
	var opts = struct {
		Command struct {
			G string `short:"g" default:"value"`
		} `command:"cmd" alias:"cm"`
	}{}

	assertParseSuccess(t, &opts, "cm")

	if opts.Command.G != "value" {
		t.Errorf("Expected G to be \"value\"")
	}
}

func TestSubCommandFindOptionByLongFlag(t *testing.T) {
	var opts struct {
		Testing bool `long:"testing" description:"Testing"`
	}

	var cmd struct {
		Other bool `long:"other" description:"Other"`
	}

	p := NewParser(&opts, Default)
	c, _ := p.AddCommand("command", "Short", "Long", &cmd)

	opt := c.FindOptionByLongName("other")

	if opt == nil {
		t.Errorf("Expected option, but found none")
	}

	assertString(t, opt.LongName, "other")

	opt = c.FindOptionByLongName("testing")

	if opt == nil {
		t.Errorf("Expected option, but found none")
	}

	assertString(t, opt.LongName, "testing")
}

func TestSubCommandFindOptionByShortFlag(t *testing.T) {
	var opts struct {
		Testing bool `short:"t" description:"Testing"`
	}

	var cmd struct {
		Other bool `short:"o" description:"Other"`
	}

	p := NewParser(&opts, Default)
	c, _ := p.AddCommand("command", "Short", "Long", &cmd)

	opt := c.FindOptionByShortName('o')

	if opt == nil {
		t.Errorf("Expected option, but found none")
	}

	if opt.ShortName != 'o' {
		t.Errorf("Expected 'o', but got %v", opt.ShortName)
	}

	opt = c.FindOptionByShortName('t')

	if opt == nil {
		t.Errorf("Expected option, but found none")
	}

	if opt.ShortName != 't' {
		t.Errorf("Expected 'o', but got %v", opt.ShortName)
	}
}
