package script

import (
	"testing"
)

func TestDockerNetworkMode(t *testing.T) {
	var d *Docker
	var expected string

	expected = DefaultDockerNetworkMode

	expected = DefaultDockerNetworkMode
	d = &Docker{}
	if actual := d.NetworkMode(); actual != expected {
		t.Errorf("The result is invalid. [expected: %s][actual: %s]", expected, actual)
	}

	expected = DefaultDockerNetworkMode
	d = &Docker{Net: nil}
	if actual := d.NetworkMode(); actual != expected {
		t.Errorf("The result is invalid. [expected: %s][actual: %s]", expected, actual)
	}

	expected = "bridge"
	d = &Docker{Net: &expected}
	if actual := d.NetworkMode(); actual != expected {
		t.Errorf("The result is invalid. [expected: %s][actual: %s]", expected, actual)
	}

	expected = "host"
	d = &Docker{Net: &expected}
	if actual := d.NetworkMode(); actual != expected {
		t.Errorf("The result is invalid. [expected: %s][actual: %s]", expected, actual)
	}
}
