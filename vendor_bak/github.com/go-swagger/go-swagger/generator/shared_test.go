package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileNaming(t *testing.T) {
	values := []struct{ Source, Expected string }{
		{"API test", "api"},
		{"Test API", "test_api"},
		{"Some API test", "some_api"},
	}

	for _, v := range values {
		assert.Equal(t, v.Expected, stripTestFromFileName(v.Source))
	}
}
