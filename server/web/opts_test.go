package web

import (
	"testing"
	"time"
)

func TestWithSync(t *testing.T) {
	opts := new(Options)
	WithSync(time.Minute)(opts)
	if got, want := opts.sync, time.Minute; got != want {
		t.Errorf("Want sync duration %v, got %v", want, got)
	}
}

func TestWithDir(t *testing.T) {
	opts := new(Options)
	WithDir("/tmp/www")(opts)
	if got, want := opts.path, "/tmp/www"; got != want {
		t.Errorf("Want www directory %q, got %q", want, got)
	}
}

func TestWithDocs(t *testing.T) {
	opts := new(Options)
	WithDocs("http://docs.drone.io")(opts)
	if got, want := opts.docs, "http://docs.drone.io"; got != want {
		t.Errorf("Want documentation url %q, got %q", want, got)
	}
}
