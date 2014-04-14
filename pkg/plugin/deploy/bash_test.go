package deploy

import (
	"strings"
	"testing"

	"github.com/drone/drone/pkg/build/buildfile"

	"gopkg.in/yaml.v1"
)

// emulate Build struct
type buildWithBash struct {
	Deploy *Deploy `yaml:"deploy,omitempty"`
}

var sampleYmlWithBash = `
deploy:
  bash:
    command: 'echo bash_deployed'
`

var sampleYmlWithScript = `
deploy:
  bash:
    script:
      - ./bin/deploy.sh
      - ./bin/check.sh
`

var sampleYmlWithBashAndScript = `
deploy:
  bash:
    command: ./bin/some_cmd.sh
    script:
      - ./bin/deploy.sh
      - ./bin/check.sh
`

func setUpWithBash(input string) (string, error) {
	var buildStruct buildWithBash
	err := yaml.Unmarshal([]byte(input), &buildStruct)
	if err != nil {
		return "", err
	}
	bf := buildfile.New()
	buildStruct.Deploy.Write(bf)
	return bf.String(), err
}

func TestBashDeployment(t *testing.T) {
	bscr, err := setUpWithBash(sampleYmlWithBash)
	if err != nil {
		t.Fatalf("Can't unmarshal deploy script: %s", err)
	}

	if !strings.Contains(bscr, "echo bash_deployed") {
		t.Error("Expect script to contains bash command")
	}
}

func TestBashDeploymentWithScript(t *testing.T) {
	bscr, err := setUpWithBash(sampleYmlWithScript)
	if err != nil {
		t.Fatalf("Can't unmarshal deploy script: %s", err)
	}

	if !strings.Contains(bscr, "./bin/deploy.sh") {
		t.Error("Expect script to contains bash script")
	}

	if !strings.Contains(bscr, "./bin/check.sh") {
		t.Error("Expect script to contains bash script")
	}
}

func TestBashDeploymentWithBashAndScript(t *testing.T) {
	bscr, err := setUpWithBash(sampleYmlWithBashAndScript)
	if err != nil {
		t.Fatalf("Can't unmarshal deploy script: %s", err)
	}

	if !strings.Contains(bscr, "./bin/deploy.sh") {
		t.Error("Expect script to contains bash script")
	}

	if !strings.Contains(bscr, "./bin/check.sh") {
		t.Error("Expect script to contains bash script")
	}

	if !strings.Contains(bscr, "./bin/some_cmd.sh") {
		t.Error("Expect script to contains bash script")
	}
}
