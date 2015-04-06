package model

import (
	"testing"
)

func Test_CreateGravatar(t *testing.T) {
	var got, want = CreateGravatar("dr_cooper@caltech.edu"), "2b77ba83e2216ddcd11fe8c24b70c2a3"
	if got != want {
		t.Errorf("Got gravatar hash %s, want %s", got, want)
	}
}

func Test_GenerateToken(t *testing.T) {
	token := GenerateToken()
	if len(token) != length {
		t.Errorf("Want token length %d, got %d", length, len(token))
	}
}
