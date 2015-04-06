package publish

import (
	"strings"
	"testing"

	"github.com/drone/drone/shared/build/buildfile"
	"github.com/drone/drone/shared/build/repo"
	"gopkg.in/yaml.v1"
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

// DockerHost and version test (no auth)
var dockerHostYaml = `
publish:
  docker:
    docker_host: tcp://server:1000
    docker_version: 1.3.0
    image_name: registry/image
`

func TestDockerHost(t *testing.T) {
	response, err := setUpWithDrone(dockerHostYaml)
	t.Log(dockerHostYaml)
	if err != nil {
		t.Fatalf("Can't unmarshal script: %s\n\n", err.Error())
	}
	expected := "export DOCKER_HOST=tcp://server:1000"
	if !strings.Contains(response, expected) {
		t.Fatalf("Response: " + response + " doesn't export correct " +
			"DOCKER_HOST envvar: expected " + expected + "\n\n")
	}
	expected = "https://get.docker.io/builds/Linux/x86_64/docker-1.3.0.tgz"
	if !strings.Contains(response, expected) {
		t.Fatalf("Response: " + response + " doesn't download from:" + expected + "\n\n")
	}
}

var dockerHostNoVersionYaml = `
publish:
  docker:
    docker_host: tcp://server:1000
    image_name: registry/image
`

func TestDockerHostNoVersion(t *testing.T) {
	response, err := setUpWithDrone(dockerHostNoVersionYaml)
	t.Log(dockerHostNoVersionYaml)
	if err != nil {
		t.Fatalf("Can't unmarshal script: %s\n\n", err.Error())
	}
	expected := "export DOCKER_HOST=tcp://server:1000"
	if !strings.Contains(response, expected) {
		t.Fatalf("Response: " + response + " doesn't export correct " +
			"DOCKER_HOST envvar: expected " + expected + "\n\n")
	}
	download := "https://get.docker.io/builds/Linux/x86_64/docker-latest.tgz"
	if !strings.Contains(response, download) {
		t.Fatalf("Response: " + response + " doesn't download from:" + download + "\n\n")
	}
}

// Private Registry Test (no auth)
var privateRegistryNoAuthYaml = `
publish:
  docker:
    dockerfile: file_path
    docker_host: tcp://server:1000
    docker_version: 1.0
    registry_login: false
    image_name: registry/image
`

func TestPrivateRegistryNoAuth(t *testing.T) {
	response, err := setUpWithDrone(privateRegistryNoAuthYaml)
	t.Log(privateRegistryNoAuthYaml)
	if err != nil {
		t.Fatalf("Can't unmarshal script: %s\n\n", err.Error())
	}
	if !strings.Contains(response, "docker build --pull -t registry/image:$(git rev-parse --short HEAD)") {
		t.Fatalf("Response: " + response + " doesn't contain registry in image-names: expected registry/image\n\n")
	}
}

// Private Registry Test (with auth)
var privateRegistryAuthYaml = `
publish:
  docker:
    dockerfile: file_path
    docker_host: tcp://server:1000
    docker_version: 1.0
    registry_login_url: https://registry:8000/v1/
    registry_login: true
    username: username
    password: password
    email: email@example.com
    image_name: registry/image
`

func TestPrivateRegistryAuth(t *testing.T) {
	response, err := setUpWithDrone(privateRegistryAuthYaml)
	t.Log(privateRegistryAuthYaml)
	if err != nil {
		t.Fatalf("Can't unmarshal script: %s\n\n", err.Error())
	}
	if !strings.Contains(response, "docker login -u username -p password -e email@example.com https://registry:8000/v1/") {
		t.Log("\n\n\n\ndocker login -u username -p xxxxxxxx -e email@example.com https://registry:8000/v1/\n\n\n\n")
		t.Fatalf("Response: " + response + " doesn't contain private registry login\n\n")
	}
	if !strings.Contains(response, "docker build --pull -t registry/image:$(git rev-parse --short HEAD) .") {
		t.Log("docker build --pull -t registry/image:$(git rev-parse --short HEAD) .")
		t.Fatalf("Response: " + response + " doesn't contain registry in image-names\n\n")
	}
}

// Keep builds Test
var keepBuildsYaml = `
publish:
  docker:
    docker_host: tcp://server:1000
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
	if strings.Contains(response, "docker rmi") {
		t.Fatalf("Response: " + response + " incorrectly instructs the docker server to remove the builds when it shouldn't\n\n")
	}
}

// Custom Tag test
var customTagYaml = `
publish:
  docker:
    docker_host: tcp://server:1000
    docker_version: 1.0
    tag: release-0.1
    username: username
    password: password
    email: email@example.com
    image_name: username/image
`

func TestSingleTag(t *testing.T) {
	response, err := setUpWithDrone(customTagYaml)
	t.Log(customTagYaml)
	if err != nil {
		t.Fatalf("Can't unmarshal script: %s\n", err.Error())
	}
	if strings.Contains(response, "$(git rev-parse --short HEAD)") {
		t.Fatalf("Response: " + response + " is tagging images from git-refs when it should use a custom tag\n\n")
	}
	if !strings.Contains(response, "docker build --pull -t username/image:release-0.1") {
		t.Fatalf("Response: " + response + " isn't tagging images using our custom tag\n\n")
	}
	if !strings.Contains(response, "docker push username/image:release-0.1") {
		t.Fatalf("Response: " + response + " doesn't push the custom tagged image\n\n")
	}
	if !strings.Contains(response, "docker rmi username/image:release-0.1") {
		t.Fatalf("Response: " + response + " doesn't remove custom tagged image\n\n")
	}
}

var missingFieldsYaml = `
publish:
  docker:
    dockerfile: file
`

var multipleTagsYaml = `
publish:
  docker:
    docker_host: tcp://server:1000
    docker_version: 1.0
    tags: [release-0.2, release-latest]
    username: username
    password: password
    email: email@example.com
    image_name: username/image
`

func TestTagsNoSingle(t *testing.T) {
	response, err := setUpWithDrone(multipleTagsYaml)
	t.Log(multipleTagsYaml)
	if err != nil {
		t.Fatalf("Can't unmarshal script: %s\n", err.Error())
	}
	if strings.Contains(response, "$(git rev-parse --short HEAD)") {
		t.Fatalf("Response: " + response + " is tagging images from git-refs when it should using custom tag\n\n")
	}
	if !strings.Contains(response, "docker build --pull -t username/image:release-0.2") {
		t.Fatalf("Response: " + response + " isn't tagging images using our first custom tag\n\n")
	}
	if !strings.Contains(response, "docker tag  username/image:release-0.2 username/image:release-latest") {
		t.Fatalf("Response: " + response + " isn't tagging images using our second custom tag\n\n")
	}
	if !strings.Contains(response, "docker push username/image:release-0.2") {
		t.Fatalf("Response: " + response + " doesn't push the custom tagged image\n\n")
	}
	if !strings.Contains(response, "docker rmi username/image:release-0.2") {
		t.Fatalf("Response: " + response + " doesn't remove custom tagged image\n\n")
	}
	if !strings.Contains(response, "docker push username/image:release-latest") {
		t.Fatalf("Response: " + response + " doesn't push the second custom tagged image\n\n")
	}
	if !strings.Contains(response, "docker rmi username/image:release-latest") {
		t.Fatalf("Response: " + response + " doesn't remove second custom tagged image\n\n")
	}
}

var bothTagsYaml = `
publish:
  docker:
    docker_host: tcp://server:1000
    docker_version: 1.0
    tag: release-0.2
    tags: [release-0.3, release-latest]
    username: username
    password: password
    email: email@example.com
    image_name: username/image
`

func TestTagsWithSingle(t *testing.T) {
	response, err := setUpWithDrone(bothTagsYaml)
	t.Log(bothTagsYaml)
	if err != nil {
		t.Fatalf("Can't unmarshal script: %s\n", err.Error())
	}
	if strings.Contains(response, "$(git rev-parse --short HEAD)") {
		t.Fatalf("Response: " + response + " is tagging images from git-refs when it should using custom tag\n\n")
	}
	if !strings.Contains(response, "docker build --pull -t username/image:release-0.3") {
		t.Fatalf("Response: " + response + " isn't tagging images using our first custom tag\n\n")
	}
	if !strings.Contains(response, "docker tag  username/image:release-0.3 username/image:release-0.2") {
		t.Fatalf("Response: " + response + " isn't tagging images using our second custom tag\n\n")
	}
	if !strings.Contains(response, "docker tag  username/image:release-0.3 username/image:release-latest") {
		t.Fatalf("Response: " + response + " isn't tagging images using our third custom tag\n\n")
	}
	if !strings.Contains(response, "docker push username/image:release-0.2") {
		t.Fatalf("Response: " + response + " doesn't push the custom tagged image\n\n")
	}
	if !strings.Contains(response, "docker rmi username/image:release-0.2") {
		t.Fatalf("Response: " + response + " doesn't remove custom tagged image\n\n")
	}
	if !strings.Contains(response, "docker push username/image:release-0.3") {
		t.Fatalf("Response: " + response + " doesn't push the custom tagged image\n\n")
	}
	if !strings.Contains(response, "docker rmi username/image:release-0.3") {
		t.Fatalf("Response: " + response + " doesn't remove custom tagged image\n\n")
	}
	if !strings.Contains(response, "docker push username/image:release-latest") {
		t.Fatalf("Response: " + response + " doesn't push the second custom tagged image\n\n")
	}
	if !strings.Contains(response, "docker rmi username/image:release-latest") {
		t.Fatalf("Response: " + response + " doesn't remove second custom tagged image\n\n")
	}
}

func TestMissingFields(t *testing.T) {
	response, err := setUpWithDrone(missingFieldsYaml)
	t.Log(missingFieldsYaml)
	if err != nil {
		t.Fatalf("Can't unmarshal script: %s\n\n", err.Error())
	}
	if !strings.Contains(response, "Missing argument(s)") {
		t.Fatalf("Response: " + response + " didn't contain missing arguments warning\n\n")
	}
}

var validYaml = `
publish:
  docker:
    docker_file: file_path
    docker_host: tcp://server:1000
    docker_version: 1.0
    username: user
    password: password
    email: email
    image_name: user/image
    registry_login: true
`

func TestValidYaml(t *testing.T) {
	response, err := setUpWithDrone(validYaml)
	t.Log(validYaml)
	if err != nil {
		t.Fatalf("Can't unmarshal script: %s\n\n", err.Error())
	}

	if !strings.Contains(response, "docker build --pull -t user/image:$(git rev-parse --short HEAD) - <") {
		t.Fatalf("Response: " + response + "doesn't contain build command for commit hash\n\n")
	}
	if !strings.Contains(response, "docker login -u user -p password -e email") {
		t.Fatalf("Response: " + response + " doesn't contain login command\n\n")
	}
	if !strings.Contains(response, "docker push user/image:$(git rev-parse --short HEAD)") {
		t.Fatalf("Response: " + response + " doesn't contain push command\n\n")
	}
	if !strings.Contains(response, "docker rmi user/image:"+
		"$(git rev-parse --short HEAD)") {
		t.Fatalf("Response: " + response + " doesn't contain remove image command\n\n")
	}
}

var withoutDockerFileYaml = `
publish:
  docker:
    docker_host: tcp://server:1000
    docker_version: 1.0
    image_name: user/image
    username: user
    password: password
    email: email
`

func TestWithoutDockerFile(t *testing.T) {
	response, err := setUpWithDrone(withoutDockerFileYaml)
	t.Log(withoutDockerFileYaml)
	if err != nil {
		t.Fatalf("Can't unmarshal script: %s\n\n", err.Error())
	}

	if !strings.Contains(response, "docker build --pull -t user/image:$(git rev-parse --short HEAD) .") {
		t.Fatalf("Response: " + response + " doesn't contain build command\n\n")
	}
}
