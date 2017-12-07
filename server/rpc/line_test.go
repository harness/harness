package rpc

import (
	"testing"
)

func TestLine(t *testing.T) {
	line := Line{
		Proc: "redis",
		Time: 60,
		Pos:  1,
		Out:  "starting redis server",
	}
	got, want := line.String(), "[redis:L1:60s] starting redis server"
	if got != want {
		t.Errorf("Wanted line string %q, got %q", want, got)
	}
}
