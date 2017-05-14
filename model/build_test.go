package model

import (
	"crypto/rand"
	"fmt"
	"testing"
)

func TestBuildTrim(t *testing.T) {
	d := make([]byte, 1000)
	rand.Read(d)

	b := Build{}
	b.Message = fmt.Sprintf("%X", d)

	if len(b.Message) != 2000 {
		t.Errorf("Failed to generate 2000 byte test string")
	}
	b.Trim()
	if len(b.Message) != 500 {
		t.Errorf("Failed to trim text string to 500 bytes")
	}
}
