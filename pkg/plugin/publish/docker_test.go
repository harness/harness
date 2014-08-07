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


// Private Registry Test (no auth)
var privateRegistryNoAuthYaml = `
publish:
  docker:
    dockerfile: file_path
    docker_server: server
    docker_port: 1000
    docker_version: 1.0
    registry_host: registry
    registry_login: false
    image_name: image
`
func TestPrivateRegistryNoAuth(t *testing.T) {
	response, err := setUpWithDrone(privateRegistryNoAuthYaml)
    if err != nil {
        t.Fatalf("Can't unmarshal script: %s\n\n", err.Error())
    }
    if !strings.Contains(response, "docker -H server:1000 build -t registry/image:$(git rev-parse --short HEAD)") {
        t.Fatalf("Response: " + response + " doesn't contain registry in image-names: expected registry/image\n\n")
    }
}

// Private Registry Test (with auth)
var privateRegistryAuthYaml = `
publish:
  docker:
    dockerfile: file_path
    docker_server: server
    docker_port: 1000
    docker_version: 1.0
    registry_host: registry
    registry_protocol: https
    registry_port: 8000
    registry_login: true
    username: username
    password: password
    email: email@example.com
    image_name: image
`
func TestPrivateRegistryAuth(t *testing.T) {
	response, err := setUpWithDrone(privateRegistryAuthYaml)
    t.Log(privateRegistryAuthYaml)
    if err != nil {
        t.Fatalf("Can't unmarshal script: %s\n\n", err.Error())
    }
    if !strings.Contains(response, "docker -H server:1000 login -u username -p password -e email@example.com https://registry:8000/v1/") {
        t.Log("\n\n\n\ndocker -H server:1000 login -u username -p xxxxxxxx -e email@example.com https://registry:8000/v1/\n\n\n\n")
		t.Fatalf("Response: " + response + " doesn't contain private registry login\n\n")
	}
    if !strings.Contains(response, "docker -H server:1000 build -t registry/image:$(git rev-parse --short HEAD) .") {
        t.Log("docker -H server:1000 build -t registry/image:$(git rev-parse --short HEAD) .")
        t.Fatalf("Response: " + response + " doesn't contain registry in image-names\n\n")
    }
}

// Override "latest" Test
var overrideLatestTagYaml = `
publish:
  docker:
    docker_server: server
    docker_port: 1000
    docker_version: 1.0
    username: username
    password: password
    email: email@example.com
    image_name: image
    push_latest: true
`
func TestOverrideLatestTag(t *testing.T) {
	response, err := setUpWithDrone(overrideLatestTagYaml)
    t.Log(overrideLatestTagYaml)
    if err != nil {
        t.Fatalf("Can't unmarshal script: %s\n\n", err.Error())
    }
    if !strings.Contains(response, "docker -H server:1000 build -t username/image:$(git rev-parse --short HEAD) .") {
        t.Fatalf("Response: " + response + " doesn't contain the git-ref tagged image\n\n")
    }
    if !strings.Contains(response, "docker -H server:1000 tag username/image:$(git rev-parse --short HEAD) username/image:latest") {
        t.Fatalf("Response: " + response + " doesn't contain 'latest' tag command\n\n")
    }
}

// Keep builds Test
var keepBuildsYaml = `
publish:
  docker:
    docker_server: server
    docker_port: 1000
    docker_version: 1.0
    keep_build: true
    username: username
    password: password
    email: email@example.com
    image_name: image
`
func TestKeepBuilds(t *testing.T) {
	response, err := setUpWithDrone(keepBuildsYaml)
    t.Log(keepBuildsYaml)
	if err != nil {
		t.Fatalf("Can't unmarshal script: %s\n\n", err.Error())
	}
	if strings.Contains(response, "docker -H server:1000 rmi") {
		t.Fatalf("Response: " + response + " incorrectly instructs the docker server to remove the builds when it shouldn't\n\n")
	}
}

// Custom Tag test
var customTagYaml = `
publish:
  docker:
    docker_server: server
    docker_port: 1000
    docker_version: 1.0
    custom_tag: release-0.1
    username: username
    password: password
    email: email@example.com
    image_name: image
`
func TestCustomTag(t *testing.T) {
	response, err := setUpWithDrone(customTagYaml)
    t.Log(customTagYaml)
    if err != nil {
		t.Fatalf("Can't unmarshal script: %s\n", err.Error())
	}
	if strings.Contains(response, "$(git rev-parse --short HEAD)") {
		t.Fatalf("Response: " + response + " is tagging images from git-refs when it should use a custom tag\n\n")
	}
	if !strings.Contains(response, "docker -H server:1000 build -t username/image:release-0.1") {
		t.Fatalf("Response: " + response + " isn't tagging images using our custom tag\n\n")
	}
	if !strings.Contains(response, "docker -H server:1000 push username/image"){
		t.Fatalf("Response: " + response + " doesn't push the custom tagged image\n\n")
	}
}

var missingFieldsYaml = `
publish:
  docker:
    dockerfile: file
`

func TestMissingFields(t *testing.T) {
    response, err := setUpWithDrone(missingFieldsYaml)
    t.Log(missingFieldsYaml)
    if err != nil {
        t.Fatalf("Can't unmarshal script: %s\n\n", err.Error())
    }
    if !strings.Contains(response, "echo \"Docker Plugin: Missing argument(s)\"") {
        t.Fatalf("Response: " + response + " didn't contain missing arguments warning\n\n")
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
    image_name: image
    push_latest: true
`

func TestValidYaml(t *testing.T) {
    response, err := setUpWithDrone(validYaml)
    t.Log(validYaml)
    if err != nil {
        t.Fatalf("Can't unmarshal script: %s\n\n", err.Error())
    }

    if !strings.Contains(response, "docker -H server:1000 tag user/image:$(git rev-parse --short HEAD) user/image:latest") {
        t.Fatalf("Response: " + response + " doesn't contain tag command for latest\n\n")
    }
    if !strings.Contains(response, "docker -H server:1000 build -t user/image:$(git rev-parse --short HEAD) - <") {
        t.Fatalf("Response: " + response + "doesn't contain build command for commit hash\n\n")
    }
    if !strings.Contains(response, "docker -H server:1000 login -u user -p password -e email") {
        t.Fatalf("Response: " + response + " doesn't contain login command\n\n")
    }
    if !strings.Contains(response, "docker -H server:1000 push user/image") {
        t.Fatalf("Response: " + response + " doesn't contain push command\n\n")
    }
    if !strings.Contains(response, "docker -H server:1000 rmi user/image:" +
        "$(git rev-parse --short HEAD)") {
        t.Fatalf("Response: " + response + " doesn't contain remove image command\n\n")
    }
}

var withoutDockerFileYaml = `
publish:
  docker:
    docker_server: server
    docker_port: 1000
    docker_version: 1.0
    image_name: image
    username: user
    password: password
    email: email
`

func TestWithoutDockerFile(t *testing.T) {
	response, err := setUpWithDrone(withoutDockerFileYaml)
	if err != nil {
		t.Fatalf("Can't unmarshal script: %s\n\n", err.Error())
	}

	if !strings.Contains(response, "docker -H server:1000 build -t user/image:$(git rev-parse --short HEAD) .") {
		t.Fatalf("Response: " + response + " doesn't contain build command\n\n")
	}
}
