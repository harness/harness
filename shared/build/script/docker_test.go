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


func TestDockerUser(t *testing.T) {
	var d *Docker
	var expected string

	expected = DefaultDockerUser
	d = &Docker{}
	if actual := d.GetUser(); actual != expected {
		t.Errorf("The result is invalid. [expected: %s][actual: %s]", expected, actual)
	}

	expected = DefaultDockerUser
	d = &Docker{User: nil}
	if actual := d.GetUser(); actual != expected {
		t.Errorf("The result is invalid. [expected: %s][actual: %s]", expected, actual)
	}

	expected = "teamcity"
	d = &Docker{User: &expected}
	if actual := d.GetUser(); actual != expected {
		t.Errorf("The result is invalid. [expected: %s][actual: %s]", expected, actual)
	}
}

func TestDockerHome(t *testing.T) {
	var d *Docker
	var expected string

	expected = DefaultDockerHome
	d = &Docker{}
	if actual := d.GetHome(); actual != expected {
		t.Errorf("The result is invalid. [expected: %s][actual: %s]", expected, actual)
	}

	expected = DefaultDockerHome
	d = &Docker{Home: nil}
	if actual := d.GetHome(); actual != expected {
		t.Errorf("The result is invalid. [expected: %s][actual: %s]", expected, actual)
	}

	expected = "/home/teamcity"
	d = &Docker{Home: &expected}
	if actual := d.GetHome(); actual != expected {
		t.Errorf("The result is invalid. [expected: %s][actual: %s]", expected, actual)
	}

	var user string = "teamcity"
	d = &Docker{User: &user}
	if actual := d.GetHome(); actual != expected {
		t.Errorf("The result is invalid. [expected: %s][actual: %s]", expected, actual)
	}
}
