package deploy

import (
	"strings"
	"testing"

	"github.com/drone/drone/pkg/build/buildfile"

	"launchpad.net/goyaml"
)

// emulate Build struct
type build struct {
	Deploy *Deploy `yaml:"deploy,omitempty"`
}

var sampleYml = `
deploy:
  ssh:
    target: user@test.example.com
    cmd: /opt/bin/redeploy.sh
`

var sampleYml1 = `
deploy:
  ssh:
    target: user@test.example.com:/srv/app/location 2212
    artifacts:
      - build.result
    cmd: /opt/bin/redeploy.sh
`

var sampleYml2 = `
deploy:
  ssh:
    target: user@test.example.com:/srv/app/location 2212
    artifacts:
      - build.result
      - config/file
    cmd: /opt/bin/redeploy.sh
`

var sampleYml3 = `
deploy:
  ssh:
    target: user@test.example.com:/srv/app/location 2212
    artifacts:
      - GITARCHIVE
    cmd: /opt/bin/redeploy.sh
`

func setUp(input string) (string, error) {
	var buildStruct build
	err := goyaml.Unmarshal([]byte(input), &buildStruct)
	if err != nil {
		return "", err
	}
	bf := buildfile.New()
	buildStruct.Deploy.Write(bf)
	return bf.String(), err
}

func TestSSHNoArtifact(t *testing.T) {
	bscr, err := setUp(sampleYml)
	if err != nil {
		t.Fatalf("Can't unmarshal deploy script: %s", err)
	}

	if strings.Contains(bscr, `scp`) {
		t.Error("Expect script not to contains scp command")
	}

	if !strings.Contains(bscr, "ssh -o StrictHostKeyChecking=no -p 22 user@test.example.com /opt/bin/redeploy.sh") {
		t.Error("Expect script to contains ssh command")
	}
}

func TestSSHOneArtifact(t *testing.T) {
	bscr, err := setUp(sampleYml1)
	if err != nil {
		t.Fatalf("Can't unmarshal deploy script: %s", err)
	}

	if !strings.Contains(bscr, "ARTIFACT=build.result") {
		t.Errorf("Expect script to contains artifact")
	}

	if !strings.Contains(bscr, "scp -o StrictHostKeyChecking=no -P 2212 -r ${ARTIFACT} user@test.example.com:/srv/app/location") {
		t.Errorf("Expect script to contains scp command, got:\n%s", bscr)
	}
}

func TestSSHMultiArtifact(t *testing.T) {
	bscr, err := setUp(sampleYml2)
	if err != nil {
		t.Fatalf("Can't unmarshal deploy script: %s", err)
	}

	if !strings.Contains(bscr, "ARTIFACT=${PWD##*/}.tar.gz") {
		t.Errorf("Expect script to contains artifact")
	}

	if !strings.Contains(bscr, "tar -cf ${ARTIFACT} build.result config/file") {
		t.Errorf("Expect script to contains tar command. got:\n", bscr)
	}
}

func TestSSHGitArchive(t *testing.T) {
	bscr, err := setUp(sampleYml3)
	if err != nil {
		t.Fatalf("Can't unmarshal deploy script: %s", err)
	}

	if !strings.Contains(bscr, "COMMIT=$(git rev-parse HEAD)") {
		t.Errorf("Expect script to contains commit ref")
	}

	if !strings.Contains(bscr, "ARTIFACT=${PWD##*/}-${COMMIT}.tar.gz") {
		t.Errorf("Expect script to contains artifact")
	}

	if strings.Contains(bscr, "=GITARCHIVE") {
		t.Errorf("Doesn't expect script to contains GITARCHIVE literals")
	}

	if !strings.Contains(bscr, "git archive --format=tar.gz --prefix=${PWD##*/}/ ${COMMIT} > ${ARTIFACT}") {
		t.Errorf("Expect script to run git archive")
	}
}
