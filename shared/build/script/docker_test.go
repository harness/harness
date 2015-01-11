package script

import (
	"testing"
)

func TestDockerNetworkMode(t *testing.T) {
	var d *Docker
	var expected string

	expected = DefaultDockerNetworkMode
	d = nil
	if actual := DockerNetworkMode(d); actual != expected {
		t.Errorf("The result is invalid. [expected: %s][actual: %s]", expected, actual)
	}

	expected = DefaultDockerNetworkMode
	d = &Docker{}
	if actual := DockerNetworkMode(d); actual != expected {
		t.Errorf("The result is invalid. [expected: %s][actual: %s]", expected, actual)
	}

	expected = DefaultDockerNetworkMode
	d = &Docker{NetworkMode: nil}
	if actual := DockerNetworkMode(d); actual != expected {
		t.Errorf("The result is invalid. [expected: %s][actual: %s]", expected, actual)
	}

	expected = "bridge"
	d = &Docker{NetworkMode: &expected}
	if actual := DockerNetworkMode(d); actual != expected {
		t.Errorf("The result is invalid. [expected: %s][actual: %s]", expected, actual)
	}

	expected = "host"
	d = &Docker{NetworkMode: &expected}
	if actual := DockerNetworkMode(d); actual != expected {
		t.Errorf("The result is invalid. [expected: %s][actual: %s]", expected, actual)
	}
}

func TestDockerHostname(t *testing.T) {
	var d *Docker
	var expected string

	expected = ""
	d = nil
	if actual := DockerHostname(d); actual != expected {
		t.Errorf("The result is invalid. [expected: %s][actual: %s]", expected, actual)
	}

	expected = ""
	d = &Docker{}
	if actual := DockerHostname(d); actual != expected {
		t.Errorf("The result is invalid. [expected: %s][actual: %s]", expected, actual)
	}

	expected = ""
	d = &Docker{Hostname: nil}
	if actual := DockerHostname(d); actual != expected {
		t.Errorf("The result is invalid. [expected: %s][actual: %s]", expected, actual)
	}

	expected = "host"
	d = &Docker{Hostname: &expected}
	if actual := DockerHostname(d); actual != expected {
		t.Errorf("The result is invalid. [expected: %s][actual: %s]", expected, actual)
	}
}
