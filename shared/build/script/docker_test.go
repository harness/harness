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

func TestIsVolumeValid(t *testing.T) {
	if IsVolumeValid("asdf:/asdf") {
		t.Error("The result is invalid.")
	}
	if IsVolumeValid("/asdf:asdf") {
		t.Error("The result is invalid.")
	}
	if !IsVolumeValid("/asdf:/asdf") {
		t.Error("The result is invalid.")
	}
	if IsVolumeValid("/asdf/asdf") {
		t.Error("The result is invalid.")
	}
	if IsVolumeValid("/asdf:/asdf:/asdf") {
		t.Error("The result is invalid.")
	}
}

func TestDockerVolumes(t *testing.T) {
	var d *Docker
	var expected []string

	expected = []string{}
	d = nil
	if actual := DockerVolumes(d); len(actual) != len(expected) {
		t.Errorf("The result is invalid. [expected: %s][actual: %s]", expected, actual)
	}

	expected = []string{}
	d = &Docker{}
	if actual := DockerVolumes(d); len(actual) != len(expected) {
		t.Errorf("The result is invalid. [expected: %s][actual: %s]", expected, actual)
	}

	expected = []string{}
	d = &Docker{Volumes: nil}
	if actual := DockerVolumes(d); len(actual) != len(expected) {
		t.Errorf("The result is invalid. [expected: %s][actual: %s]", expected, actual)
	}

	expected = []string{}
	d = &Docker{Volumes: expected}
	if actual := DockerVolumes(d); len(actual) != len(expected) {
		t.Errorf("The result is invalid. [expected: %s][actual: %s]", expected, actual)
	}

	expected = []string{"/test/123:/test/123"}
	d = &Docker{Volumes: expected}
	if actual := DockerVolumes(d); len(actual) != len(expected) {
		t.Errorf("The result is invalid. [expected: %s][actual: %s]", expected, actual)
	}
}
