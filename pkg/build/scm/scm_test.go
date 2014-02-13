package scm

import (
	"testing"
)

func TestGitDepth(t *testing.T) {
	var s *Scm = nil
	if d := GitDepth(s); d != DefaultGitDepth {
		t.Errorf("The result is invalid. [expected: %d][actual: %d]", DefaultGitDepth, d)
	}

	s = &Scm{}
	if d := GitDepth(s); d != DefaultGitDepth {
		t.Errorf("The result is invalid. [expected: %d][actual: %d]", DefaultGitDepth, d)
	}

	s = &Scm{Git: nil}
	if d := GitDepth(s); d != DefaultGitDepth {
		t.Errorf("The result is invalid. [expected: %d][actual: %d]", DefaultGitDepth, d)
	}

	s = &Scm{Git: &Git{}}
	if d := GitDepth(s); d != DefaultGitDepth {
		t.Errorf("The result is invalid. [expected: %d][actual: %d]", DefaultGitDepth, d)
	}

	s = &Scm{Git: &Git{Depth: ""}}
	if d := GitDepth(s); d != DefaultGitDepth {
		t.Errorf("The result is invalid. [expected: %d][actual: %d]", DefaultGitDepth, d)
	}

	s = &Scm{Git: &Git{Depth: "a"}}
	if d := GitDepth(s); d != DefaultGitDepth {
		t.Errorf("The result is invalid. [expected: %d][actual: %d]", DefaultGitDepth, d)
	}

	s = &Scm{Git: &Git{Depth: "0"}}
	if d := GitDepth(s); d != 0 {
		t.Errorf("The result is invalid. [expected: %d][actual: %d]", DefaultGitDepth, 0)
	}

	s = &Scm{Git: &Git{Depth: "1"}}
	if d := GitDepth(s); d != 1 {
		t.Errorf("The result is invalid. [expected: %d][actual: %d]", DefaultGitDepth, 1)
	}
}
