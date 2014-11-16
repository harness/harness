package deploy

import (
	"strings"
	"testing"

	"github.com/drone/drone/shared/build/buildfile"

	"gopkg.in/yaml.v1"
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
    artifacts: ./
    cmd: /opt/bin/redeploy.sh
`

var sampleYml2 = `
deploy:
  ssh:
    target: user@test.example.com:/srv/app/location 2212
    artifacts: |
      build.result
      config/file
    cmd: /opt/bin/redeploy.sh
`

var sampleYml3 = `
deploy:
  ssh:
    target: user@test.example.com:/srv/app/location 2212
    artifacts: ./
    cmd: |
      cd /srv/app/location
      bundle install --deployment
      bundle exec rake assets:clobber
      RAILS_ENV=production bundle exec rake assets:precompile
      sudo sv restart puma
`

func setUp(input string) (string, error) {
	var buildStruct build
	err := yaml.Unmarshal([]byte(input), &buildStruct)
	if err != nil {
		return "", err
	}
	bf := buildfile.New()
	buildStruct.Deploy.Write(bf, nil)
	return bf.String(), err
}

func TestSSHNoArtifact(t *testing.T) {
	bscr, err := setUp(sampleYml)
	if err != nil {
		t.Fatalf("Can't unmarshal deploy script: %s", err)
	}

	if strings.Contains(bscr, "rsync") {
		t.Error("Expect script not to contains rsync command")
	}

	if !strings.Contains(bscr, "ssh -o StrictHostKeyChecking=no -p 22 user@test.example.com") {
		t.Error("Expect script to contains ssh command")
	}
}

func TestSSHOneArtifact(t *testing.T) {
	bscr, err := setUp(sampleYml1)
	if err != nil {
		t.Fatalf("Can't unmarshal deploy script: %s", err)
	}

	if !strings.Contains(bscr, `printf "./" > ${ARTIFACTS}`) {
		t.Errorf("Expect script to contains artifacts file listing")
	}

	if !strings.Contains(bscr, `rsync -avze 'ssh -p 2212' --files-from ${ARTIFACTS} ./ user@test.example.com:/srv/app/location`) {
		t.Errorf("Expect script to contains rsync command, got:\n%s", bscr)
	}
}

func TestSSHMultiArtifact(t *testing.T) {
	bscr, err := setUp(sampleYml2)
	if err != nil {
		t.Fatalf("Can't unmarshal deploy script: %s", err)
	}

	if !strings.Contains(bscr, `printf "build.result\nconfig/file\n" > ${ARTIFACTS}`) {
		t.Errorf("Expect script to contains artifact")
	}
}

