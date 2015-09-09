package types

import (
	"testing"
)

func Test_GenerateToken(t *testing.T) {
	token := GenerateToken()
	if len(token) != length {
		t.Errorf("Want token length %d, got %d", length, len(token))
	}
}
