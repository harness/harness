package publish

import (
	"fmt"

	"github.com/drone/drone/plugin/condition"
	"github.com/drone/drone/shared/build/buildfile"
)

type Docker struct {
	// The path to the dockerfile to create the image from. If the path is empty or no
	// path is specified then the docker file will be built from the base directory.
	Dockerfile string `yaml:"docker_file"`

	// Connection information for the docker server that will build the image
	// Same format than DOCKER_HOST envvar, i.e.: tcp://172.16.1.1:2375
	DockerHost string `yaml:"docker_host"`
	// The Docker client version to download. This must match the docker version on the server
	DockerVersion string `yaml:"docker_version"`

	// Optional Arguments to allow finer-grained control of registry
	// endpoints
	RegistryLoginUrl string `yaml:"registry_login_url"`
	ImageName        string `yaml:"image_name"`
	RegistryLogin    bool   `yaml:"registry_login"`

	// Authentication credentials for index.docker.io
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Email    string `yaml:"email"`

	// Keep the build on the Docker host after pushing?
	KeepBuild bool `yaml:"keep_build"`
	// Do we want to override "latest" automatically with this build?
	PushLatest bool   `yaml:"push_latest"`
	CustomTag  string `yaml:"custom_tag"`

	Condition *condition.Condition `yaml:"when,omitempty"`
}

// Write adds commands to the buildfile to do the following:
// 1. Install the docker client in the Drone container.
// 2. Build a docker image based on the dockerfile defined in the config.
// 3. Push that docker image to index.docker.io.
// 4. Delete the docker image on the server it was build on so we conserve disk space.
func (d *Docker) Write(f *buildfile.Buildfile) {
	if len(d.DockerHost) == 0 || len(d.DockerVersion) == 0 || len(d.ImageName) == 0 {
		f.WriteCmdSilent(`echo -e "Docker Plugin: Missing argument(s)\n\n"`)
		if len(d.DockerHost) == 0 {
			f.WriteCmdSilent(`echo -e "\tdocker_host not defined in yaml"`)
		}
		if len(d.DockerVersion) == 0 {
			f.WriteCmdSilent(`echo -e "\tdocker_version not defined in yaml"`)
		}
		if len(d.ImageName) == 0 {
			f.WriteCmdSilent(`echo -e "\timage_name not defined in yaml"`)
		}
		return
	}
	// Download docker binary and install it as /usr/local/bin/docker
	f.WriteCmd("wget -qO- https://get.docker.io/builds/Linux/x86_64/docker-" +
		d.DockerVersion + ".tgz |sudo tar zxf - -C /")

	dockerPath := "."
	if len(d.Dockerfile) != 0 {
		dockerPath = fmt.Sprintf("- < %s", d.Dockerfile)
	}

	// Run the command commands to build and deploy the image.
	// Are we setting a custom tag, or do we use the git hash?
	imageTag := ""
	if len(d.CustomTag) > 0 {
		imageTag = d.CustomTag
	} else {
		imageTag = "$(git rev-parse --short HEAD)"
	}
	f.WriteCmd("export DOCKER_HOST=" + d.DockerHost)
	f.WriteCmd(fmt.Sprintf("docker build -t %s:%s %s", d.ImageName, imageTag, dockerPath))

	// Login?
	if d.RegistryLogin == true {
		f.WriteCmdSilent(fmt.Sprintf("docker login -u %s -p %s -e %s %s",
			d.Username, d.Password, d.Email, d.RegistryLoginUrl))
	}
	// Push the tagged image only - Do not push all image tags
	f.WriteCmd(fmt.Sprintf("docker push %s:%s", d.ImageName, imageTag))

	// Are we overriding the "latest" tag?
	if d.PushLatest {
		f.WriteCmd(fmt.Sprintf("docker tag %s:%s %s:latest",
			d.ImageName, imageTag, d.ImageName))
		f.WriteCmd(fmt.Sprintf("docker push %s:latest", d.ImageName))
	}

	// Delete the image from the docker server we built on.
	if !d.KeepBuild {
		f.WriteCmd(fmt.Sprintf("docker rmi %s:%s", d.ImageName, imageTag))
		if d.PushLatest {
			f.WriteCmd(fmt.Sprintf("docker rmi %s:latest", d.ImageName))
		}
	}
}

func (d *Docker) GetCondition() *condition.Condition {
	return d.Condition
}
