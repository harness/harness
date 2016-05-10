package dockerclient

import (
	"testing"
)

func TestAuthEncode(t *testing.T) {
	a := AuthConfig{Username: "foo", Password: "password", Email: "bar@baz.com"}
	expected := "eyJ1c2VybmFtZSI6ImZvbyIsInBhc3N3b3JkIjoicGFzc3dvcmQiLCJlbWFpbCI6ImJhckBiYXouY29tIn0K"
	got, _ := a.encode()

	if expected != got {
		t.Errorf("testAuthEncode failed. Expected [%s] got [%s]", expected, got)
	}
}
