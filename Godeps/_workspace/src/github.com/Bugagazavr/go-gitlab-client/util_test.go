package gogitlab

import (
	"github.com/drone/drone/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"testing"
)

func TestParameterEncoding(t *testing.T) {
	assert.Equal(t, encodeParameter("namespace/project"), "namespace%2Fproject")
	assert.Equal(t, encodeParameter("14"), "14")
}
