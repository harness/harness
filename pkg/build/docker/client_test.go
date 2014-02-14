package docker

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestHostFromEnv(t *testing.T) {
	os.Setenv("DOCKER_HOST", "tcp://1.1.1.1:4243")
	defer os.Setenv("DOCKER_HOST", "")

	client := New()

	if client.proto != "tcp" {
		t.Fail()
	}

	if client.addr != "1.1.1.1:4243" {
		t.Fail()
	}
}

func TestInvalidHostFromEnv(t *testing.T) {
	os.Setenv("DOCKER_HOST", "tcp:1.1.1.1:4243") // missing tcp:// prefix
	defer os.Setenv("DOCKER_HOST", "")

	client := New()

	if client.addr == "1.1.1.1:4243" {
		t.Fail()
	}
}

func TestSocketHost(t *testing.T) {
	// create temporary file to represent the docker socket
	file, err := ioutil.TempFile("", "TestDefaultUnixHost")
	if err != nil {
		t.Fail()
	}
	file.Close()
	defer os.Remove(file.Name())

	client := &Client{}
	client.setHost(file.Name())

	if client.proto != "unix" {
		t.Fail()
	}

	if client.addr != file.Name() {
		t.Fail()
	}
}

func TestDefaultTcpHost(t *testing.T) {
	client := &Client{}
	client.setHost("/tmp/missing_socket")

	if client.proto != "tcp" {
		t.Fail()
	}

	if client.addr != "0.0.0.0:4243" {
		t.Fail()
	}
}
