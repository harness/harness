package git

import (
	"testing"
)

func TestGitDepth(t *testing.T) {
	var g *Git
	var expected int

	expected = DefaultGitDepth
	g = nil
	if actual := GitDepth(g); actual != expected {
		t.Errorf("The result is invalid. [expected: %d][actual: %d]", expected, actual)
	}

	expected = DefaultGitDepth
	g = &Git{}
	if actual := GitDepth(g); actual != expected {
		t.Errorf("The result is invalid. [expected: %d][actual: %d]", expected, actual)
	}

	expected = DefaultGitDepth
	g = &Git{Depth: nil}
	if actual := GitDepth(g); actual != expected {
		t.Errorf("The result is invalid. [expected: %d][actual: %d]", expected, actual)
	}

	expected = 0
	g = &Git{Depth: &expected}
	if actual := GitDepth(g); actual != expected {
		t.Errorf("The result is invalid. [expected: %d][actual: %d]", expected, actual)
	}

	expected = 1
	g = &Git{Depth: &expected}
	if actual := GitDepth(g); actual != expected {
		t.Errorf("The result is invalid. [expected: %d][actual: %d]", expected, actual)
	}
}
