package deploy

import (
    "strings"
    "testing"

    "github.com/drone/drone/pkg/build/buildfile"

    "launchpad.net/goyaml"
)

// emulate Build struct
type DeployToCF struct {
    Deploy *Deploy `yaml:"deploy,omitempty"`
}

var sampleYmlBasic = `
deploy:
  cloudfoundry:
    target: https://api.example.com
    username: foo
    password: bar
`

var sampleYmlWithOrg = `
deploy:
  cloudfoundry:
    target: https://api.example.com
    username: foo
    password: bar
    org: custom-org
`

var sampleYmlWithSpace = `
deploy:
  cloudfoundry:
    target: https://api.example.com
    username: foo
    password: bar
    org: custom-org
    space: dev
`

var sampleYmlWithAppName = `
deploy:
  cloudfoundry:
    target: https://api.example.com
    username: foo
    password: bar
    app: test-app
`

func setUpWithCF(input string) (string, error) {
    var buildStruct DeployToCF
    err := goyaml.Unmarshal([]byte(input), &buildStruct)
    if err != nil {
        return "", err
    }
    bf := buildfile.New()
    buildStruct.Deploy.Write(bf)
    return bf.String(), err
}

func TestCloudFoundryToolInstall(t *testing.T) {
    bscr, err := setUpWithCF(sampleYmlBasic)
    if err != nil {
        t.Fatalf("Can't unmarshal deploy script: %s", err)
    }

    if !strings.Contains(bscr, "curl -sLO http://go-cli.s3-website-us-east-1.amazonaws.com/releases/latest/cf-cli_amd64.deb") {
        t.Error("Expect script to contain download command")
    }

    if !strings.Contains(bscr, "dpkg -i cf-cli_amd64.deb") {
        t.Error("Expect script to contain install command")
    }
}

func TestCloudFoundryDeployment(t *testing.T) {
    bscr, err := setUpWithCF(sampleYmlBasic)
    if err != nil {
        t.Fatalf("Can't unmarshal deploy script: %s", err)
    }

    if !strings.Contains(bscr, "cf login -a https://api.example.com -u foo -p bar") {
        t.Error("Expect login script to contain username and password")
    }

    if !strings.Contains(bscr, "cf push") {
        t.Error("Expect script to contain push")
    }
}

func TestCloudFoundryDeploymentWithOrg(t *testing.T) {
    bscr, err := setUpWithCF(sampleYmlWithOrg)
    if err != nil {
        t.Fatalf("Can't unmarshal deploy script: %s", err)
    }

    if !strings.Contains(bscr, "cf login -a https://api.example.com -u foo -p bar -o custom-org") {
        t.Error("Expect login script to contain organization")
    }
}

func TestCloudFoundryDeploymentWithSpace(t *testing.T) {
    bscr, err := setUpWithCF(sampleYmlWithSpace)
    if err != nil {
        t.Fatalf("Can't unmarshal deploy script: %s", err)
    }

    if !strings.Contains(bscr, "cf login -a https://api.example.com -u foo -p bar -o custom-org -s dev") {
        t.Error("Expect login script to contain space")
    }
}

func TestCloudFoundryDeploymentWithApp(t *testing.T) {
    bscr, err := setUpWithCF(sampleYmlWithAppName)
    if err != nil {
        t.Fatalf("Can't unmarshal deploy script: %s", err)
    }

    if !strings.Contains(bscr, "cf push test-app") {
        t.Error("Expect login script to contain app name")
    }
}
