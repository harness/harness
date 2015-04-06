package docker

import (
	"os"
	"testing"
)

func TestHostFromEnv(t *testing.T) {
	os.Setenv("DOCKER_HOST", "tcp://1.1.1.1:2375")
	defer os.Setenv("DOCKER_HOST", "")

	client := New()

	if client.proto != "tcp" {
		t.Fail()
	}

	if client.addr != "1.1.1.1:2375" {
		t.Fail()
	}
}

func TestInvalidHostFromEnv(t *testing.T) {
	os.Setenv("DOCKER_HOST", "tcp:1.1.1.1:2375") // missing tcp:// prefix
	defer os.Setenv("DOCKER_HOST", "")

	client := New()

	if client.addr == "1.1.1.1:2375" {
		t.Fail()
	}
}
