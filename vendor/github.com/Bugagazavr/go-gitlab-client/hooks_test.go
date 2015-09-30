package gogitlab

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHook(t *testing.T) {
	ts, gitlab := Stub("stubs/hooks/show.json")
	hook, err := gitlab.ProjectHook("1", "2")

	assert.Equal(t, err, nil)
	assert.IsType(t, new(Hook), hook)
	assert.Equal(t, hook.Url, "http://example.com/hook")
	defer ts.Close()
}
