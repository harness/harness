package docker

import (
	"testing"
)

func Test_toContainerConfig(t *testing.T) {
	t.Skip()
}

func Test_toAuthConfig(t *testing.T) {
	t.Skip()
}

func Test_toEnvironmentSlice(t *testing.T) {
	env := map[string]string{
		"HOME": "/root",
	}
	envs := toEnvironmentSlice(env)
	want, got := "HOME=/root", envs[0]
	if want != got {
		t.Errorf("Wanted envar %s got %s", want, got)
	}
}
