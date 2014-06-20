package publish

import (
    "strings"
    "testing"

    "gopkg.in/v1/yaml"
    "github.com/drone/drone/pkg/build/buildfile"
    "github.com/drone/drone/pkg/build/repo"
)

type PublishToDrone struct {
    Publish *Publish `yaml:"publish,omitempty"`
}

func setUpWithDrone(input string) (string, error) {
    var buildStruct PublishToDrone
    err := yaml.Unmarshal([]byte(input), &buildStruct)
    if err != nil {
        return "", err
    }
    bf := buildfile.New()
    buildStruct.Publish.Write(bf, &repo.Repo{Name: "name"})
    return bf.String(), err
}

var missingFieldsYaml = `
publish:
  docker:
    dockerfile: file
`

func TestMissingFields(t *testing.T) {
    response, err := setUpWithDrone(missingFieldsYaml)
    if err != nil {
        t.Fatalf("Can't unmarshal script: %s", err.Error())
    }
    if !strings.Contains(response, "echo \"Docker Plugin: Missing argument(s)") {
        t.Fatalf("Response: " + response + " didn't contain missing arguments warning")
    }
}

var validYaml = `
publish:
  docker:
    docker_file: file_path
    docker_server: server
    docker_port: 1000
    docker_version: 1.0
    repo_base_name: base_repo
    username: user
    password: password
    email: email
`

func TestValidYaml(t *testing.T) {
    response, err := setUpWithDrone(validYaml)
    if err != nil {
        t.Fatalf("Can't unmarshal script: %s", err.Error())
    }
    if !strings.Contains(response, "docker -H server:1000 build -t base_repo/name" +
        ":$(git rev-parse --short HEAD)") {
        t.Fatalf("Response: " + response + "doesn't contain build command")
    }
    if !strings.Contains(response, "docker -H server:1000 login -u user -p password -e email") {
        t.Fatalf("Response: " + response + " doesn't contain login command")
    }
    if !strings.Contains(response, "docker -H server:1000 push base_repo/name") {
        t.Fatalf("Response: " + response + " doesn't contain push command")
    }
    if !strings.Contains(response, "docker -H server:1000 rmi base_repo/name:" +
        "$(git rev-parse --short HEAD)") {
        t.Fatalf("Response: " + response + " doesn't contain remove image command")
    }
}
