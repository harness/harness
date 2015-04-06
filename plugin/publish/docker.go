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
	// The Docker client version to download. Will default to latest if not set
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
	KeepBuild bool     `yaml:"keep_build"`
	Tag       string   `yaml:"tag"`
	Tags      []string `yaml:"tags"`
	ForceTags bool     `yaml:"force_tags"`

	Condition *condition.Condition `yaml:"when,omitempty"`
}

// Write adds commands to the buildfile to do the following:
// 1. Install the docker client in the Drone container if required.
// 2. Build a docker image based on the dockerfile defined in the config.
// 3. Push that docker image to index.docker.io.
// 4. Delete the docker image on the server it was build on so we conserve disk space.
func (d *Docker) Write(f *buildfile.Buildfile) {
	if len(d.DockerHost) == 0 || len(d.ImageName) == 0 {
		f.WriteCmdSilent(`echo -e "Docker Plugin: Missing argument(s)\n\n"`)
		if len(d.DockerHost) == 0 {
			f.WriteCmdSilent(`echo -e "\tdocker_host not defined in yaml"`)
		}
		if len(d.ImageName) == 0 {
			f.WriteCmdSilent(`echo -e "\timage_name not defined in yaml"`)
		}
		return
	}

	// If docker version is unspecified, download and install the latest client
	if len(d.DockerVersion) == 0 {
		d.DockerVersion = "latest"
	}

	if len(d.DockerVersion) > 0 {
		// Download docker binary and install it as /usr/local/bin/docker if it does not exist
		f.WriteCmd("type -p docker || wget -qO- https://get.docker.io/builds/Linux/x86_64/docker-" +
			d.DockerVersion + ".tgz |sudo tar zxf - -C /")
	}

	dockerPath := "."
	if len(d.Dockerfile) != 0 {
		dockerPath = fmt.Sprintf("- < %s", d.Dockerfile)
	}

	// Run the command commands to build and deploy the image.

	// Add the single tag if one exists
	if len(d.Tag) > 0 {
		d.Tags = append(d.Tags, d.Tag)
	}

	// If no tags are specified, use the commit hash
	if len(d.Tags) == 0 {
		d.Tags = append(d.Tags, "$(git rev-parse --short HEAD)")
	}

	// There is always at least 1 tag
	buildImageTag := d.Tags[0]

	// Export docker host once
	f.WriteCmd("export DOCKER_HOST=" + d.DockerHost)

	// Build the image
	f.WriteCmd(fmt.Sprintf("docker build --pull -t %s:%s %s", d.ImageName, buildImageTag, dockerPath))

	// Login?
	if d.RegistryLogin == true {
		f.WriteCmdSilent(fmt.Sprintf("docker login -u %s -p %s -e %s %s",
			d.Username, d.Password, d.Email, d.RegistryLoginUrl))
	}

	// Tag and push all tags
	for _, tag := range d.Tags {
		if tag != buildImageTag {
			var options string
			if d.ForceTags {
				options = "-f"
			}
			f.WriteCmd(fmt.Sprintf("docker tag %s %s:%s %s:%s", options, d.ImageName, buildImageTag, d.ImageName, tag))
		}

		f.WriteCmd(fmt.Sprintf("docker push %s:%s", d.ImageName, tag))
	}

	// Remove tags after pushing unless keepBuild is set
	if !d.KeepBuild {
		for _, tag := range d.Tags {
			f.WriteCmd(fmt.Sprintf("docker rmi %s:%s", d.ImageName, tag))
		}
	}
}

func (d *Docker) GetCondition() *condition.Condition {
	return d.Condition
}
