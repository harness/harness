package publish

import (
    "strings"
    "testing"

    "github.com/drone/drone/pkg/build/buildfile"

    "launchpad.net/goyaml"
)

// emulate Build struct
type PublishToNPM struct {
    Publish *Publish `yaml:"publish,omitempty"`
}

var sampleYml1 = `
publish:
  npm:
    username: foo
    email: foo@example.com
    password: bar
`

var sampleYml2 = `
publish:
  npm:
    username: foo
    email: foo@example.com
    password: bar
    force: true
`

var sampleYmlWithReg = `
publish:
  npm:
    username: foo
    email: foo@example.com
    password: bar
    registry: https://npm.example.com/me/
    folder: my-project/node-app/
    tag: 1.2.3
`

func setUpWithNPM(input string) (string, error) {
    var buildStruct PublishToNPM
    err := goyaml.Unmarshal([]byte(input), &buildStruct)
    if err != nil {
        return "", err
    }
    bf := buildfile.New()
    buildStruct.Publish.Write(bf, nil)
    return bf.String(), err
}

func TestNPMPublish(t *testing.T) {
    bscr, err := setUpWithNPM(sampleYml1)
    if err != nil {
        t.Fatalf("Can't unmarshal publish script: %s", err)
    }

    if !strings.Contains(bscr, "npm publish") {
        t.Error("Expect script to contain install command")
    }
}

func TestNPMForcePublish(t *testing.T) {
    bscr, err := setUpWithNPM(sampleYml2)
    if err != nil {
        t.Fatalf("Can't unmarshal publish script: %s", err)
    }

    if !strings.Contains(bscr, "npm publish  --force") {
        t.Error("Expect script to contain install command")
    }
}

func TestNPMPublishRegistry(t *testing.T) {
    bscr, err := setUpWithNPM(sampleYmlWithReg)
    if err != nil {
        t.Fatalf("Can't unmarshal publish script: %s", err)
    }

    if !strings.Contains(bscr, "npm config set registry https://npm.example.com/me/") {
        t.Error("Expect script to contain npm config registry command")
    }

    if !strings.Contains(bscr, "npm publish my-project/node-app/ --tag 1.2.3") {
        t.Error("Expect script to contain npm publish command")
    }
}
