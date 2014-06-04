package util

import (
	"testing"
)

func Test_CreateGravatar(t *testing.T) {
	var got, want = CreateGravatar("dr_cooper@caltech.edu"), "2b77ba83e2216ddcd11fe8c24b70c2a3"
	if got != want {
		t.Errorf("Got gravatar hash %s, want %s", got, want)
	}
}

func TestCreateSlug(t *testing.T) {
	var slugs = map[string]string{
		"sheldoncooper":    "sheldoncooper",
		"SheldonCooper":    "sheldoncooper",
		"Sheldon-Cooper":   "sheldon-cooper",
		"Sheldon_Cooper":   "sheldon-cooper",
		"Sheldon Cooper":   "sheldon-cooper",
		"Sheldon Cooper1":  "sheldon-cooper1",
		"Sheldon Cooper*":  "sheldon-cooper",
		"Sheldon[Cooper]":  "sheldon-cooper",
		"Sheldon[Cooper]]": "sheldon-cooper",
		// let's try almost every single special character
		"Sheldon!@#$%^&*()+=,<.>/?_Cooper": "sheldon-cooper",
		"Sheldon!@#$%^&*()+=,<->/?Cooper":  "sheldon-cooper",
	}

	for in, out := range slugs {
		var got, want = CreateSlug(in), out
		if got != want {
			t.Errorf("Got slug %s, want %s", got, want)
		}
	}
}

func TestGenerateToken(t *testing.T) {
	token := GenerateToken()
	if len(token) != length {
		t.Errorf("Want token length %d, got %d", length, len(token))
	}
}
