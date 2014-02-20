package docker

import (
	"testing"
)

func TestCopyEmptySource(t *testing.T) {
	c := ContainerService{}
	err := c.Copy("fakeid", "", "")

	if err == nil {
		t.Fail()
	}
	if err.Error() != "docker:cp source must not be empty" {
		t.Fail()
	}
}
