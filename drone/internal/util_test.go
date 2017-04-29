package internal

import "testing"

func TestParseKeyPair(t *testing.T) {
	s := []string{"FOO=bar", "BAR=", "INVALID"}
	p := ParseKeyPair(s)
	if p["FOO"] != "bar" {
		t.Errorf("Wanted %q, got %q.", "bar", p["FOO"])
	}
	if _, exists := p["BAR"]; !exists {
		t.Error("Missing a key with no value. Keys with empty values are also valid.")
	}
	if _, exists := p["INVALID"]; exists {
		t.Error("Keys without an equal sign suffix are invalid.")
	}
}
