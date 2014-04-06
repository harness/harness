package client

import (
	"testing"
	"time"
)

func TestCopy(t *testing.T) {
	l1 := &Line{
		Nick: "nick",
		Ident: "ident",
		Host: "host",
		Src: "src",
		Cmd: "cmd",
		Raw: "raw",
		Args: []string{"arg", "text"},
		Time: time.Now(),
	}

	l2 := l1.Copy()

	// Ugly. Couldn't be bothered to bust out reflect and actually think.
	if l2.Nick != "nick" || l2.Ident != "ident" || l2.Host != "host" ||
		l2.Src != "src" || l2.Cmd != "cmd" || l2.Raw != "raw" ||
		l2.Args[0] != "arg" || l2.Args[1] != "text" {
		t.Errorf("Line not copied correctly")
		t.Errorf("l1: %#v\nl2: %#v", l1, l2)
	}

	// Now, modify l2 and verify l1 not changed
	l2.Nick = l2.Nick[1:]
	l2.Ident = "foo"
	l2.Host = ""
	l2.Args[0] = l2.Args[0][1:]
	l2.Args[1] = "bar"

	if l1.Nick != "nick" || l1.Ident != "ident" || l1.Host != "host" ||
		l1.Src != "src" || l1.Cmd != "cmd" || l1.Raw != "raw" ||
		l1.Args[0] != "arg" || l1.Args[1] != "text" {
		t.Errorf("Original modified when copy changed")
		t.Errorf("l1: %#v\nl2: %#v", l1, l2)
	}
}
