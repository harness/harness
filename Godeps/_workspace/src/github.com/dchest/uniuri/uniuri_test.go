package uniuri

import "testing"

func TestNew(t *testing.T) {
	u := New()
	// Check length
	if len(u) != StdLen {
		t.Fatalf("wrong length: expected %d, got %d", StdLen, len(u))
	}
	// Check that only allowed characters are present
	for _, c := range u {
		var present bool
		for _, a := range StdChars {
			if rune(a) == c {
				present = true
			}
		}
		if !present {
			t.Fatalf("chars not allowed in %q", u)
		}
	}
	// Generate 1000 uniuris and check that they are unique
	uris := make([]string, 1000)
	for i, _ := range uris {
		uris[i] = New()
	}
	for i, u := range uris {
		for j, u2 := range uris {
			if i != j && u == u2 {
				t.Fatalf("not unique: %d:%q and %d:%q", i, j, u, u2)
			}
		}
	}
}
