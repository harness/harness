package publish

import (
	"fmt"
	"strings"
	"testing"

	"gopkg.in/yaml.v1"
)

var validcfg = map[string]interface{}{
	"artifacts": []string{"release/"},
	"tag":       "v1.0",
	"token":     "github-token",
	"user":      "drone",
	"repo":      "drone",
}

func buildfileForConfig(config map[string]interface{}) (string, error) {
	yml, err := yaml.Marshal(map[string]interface{}{
		"publish": config,
	})
	if err != nil {
		return "", err
	}
	return setUpWithDrone(string(yml))
}

func TestRequiredConfig(t *testing.T) {
	for _, required := range []string{"artifacts", "tag", "token", "user", "repo"} {
		invalidcfg := make(map[string]interface{})
		for k, v := range validcfg {
			if k != required {
				invalidcfg[k] = v
			}
		}
		buildfilestr, err := buildfileForConfig(map[string]interface{}{"github": invalidcfg})
		if err != nil {
			t.Fatal(err)
		}
		contains := fmt.Sprintf("%s not defined", required)
		if !strings.Contains(buildfilestr, contains) {
			t.Fatalf("Expected buildfile to contain error '%s': %s", contains, buildfilestr)
		}
	}
}

func TestScript(t *testing.T) {
	cmd := "echo run me!"
	scriptcfg := make(map[string]interface{})
	scriptcfg["script"] = []string{cmd}
	for k, v := range validcfg {
		scriptcfg[k] = v
	}
	buildfilestr, err := buildfileForConfig(map[string]interface{}{"github": scriptcfg})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buildfilestr, cmd) {
		t.Fatalf("Expected buildfile to contain command '%s': %s", cmd, buildfilestr)
	}
}

func TestDefaultBehavior(t *testing.T) {
	buildfilestr, err := buildfileForConfig(map[string]interface{}{"github": validcfg})
	if err != nil {
		t.Fatal(err)
	}
	defaultname := fmt.Sprintf(`-n "%s"`, validcfg["tag"].(string))
	if !strings.Contains(buildfilestr, defaultname) {
		t.Fatalf("Expected buildfile to contain name default to tag '%s': %s", defaultname, buildfilestr)
	}
	if strings.Contains(buildfilestr, "--draft") {
		t.Fatalf("Should not create a draft release by default: %s", buildfilestr)
	}
	if strings.Contains(buildfilestr, "--pre-release") {
		t.Fatalf("Should not create a pre-release release by default: %s", buildfilestr)
	}
	if !strings.Contains(buildfilestr, "github-release release") {
		t.Fatalf("Should create a release: %s", buildfilestr)
	}
	if !strings.Contains(buildfilestr, "github-release upload") {
		t.Fatalf("Should upload a file: %s", buildfilestr)
	}
}

func TestOpts(t *testing.T) {
	optscfg := make(map[string]interface{})
	optscfg["draft"] = true
	optscfg["prerelease"] = true
	for k, v := range validcfg {
		optscfg[k] = v
	}
	buildfilestr, err := buildfileForConfig(map[string]interface{}{"github": optscfg})
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"--draft", "--pre-release"} {
		if !strings.Contains(buildfilestr, flag) {
			t.Fatalf("Expected buildfile to contain flag '%s': %s", flag, buildfilestr)
		}
	}
}
