package publish

import (
	"fmt"
	"strconv"

	"github.com/drone/drone/shared/build/buildfile"
	"github.com/drone/drone/shared/build/repo"
)

type Docker struct {
	// The path to the dockerfile to create the image from. If the path is empty or no
	// path is specified then the docker file will be built from the base directory.
	Dockerfile string `yaml:"docker_file"`

	// Connection information for the docker server that will build the image
	DockerServer     string `yaml:"docker_server"`
	DockerServerPort int    `yaml:"docker_port"`
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
	Branch     string `yaml:"branch"`
}

// Write adds commands to the buildfile to do the following:
// 1. Install the docker client in the Drone container.
// 2. Build a docker image based on the dockerfile defined in the config.
// 3. Push that docker image to index.docker.io.
// 4. Delete the docker image on the server it was build on so we conserve disk space.
func (d *Docker) Write(f *buildfile.Buildfile, r *repo.Repo) {
	if len(d.DockerServer) == 0 || d.DockerServerPort == 0 || len(d.DockerVersion) == 0 ||
		len(d.ImageName) == 0 {
		f.WriteCmdSilent(`echo -e "Docker Plugin: Missing argument(s)\n\n"`)
		if len(d.DockerServer) == 0 {
			f.WriteCmdSilent(`echo -e "\tdocker_server not defined in yaml"`)
		}
		if d.DockerServerPort == 0 {
			f.WriteCmdSilent(`echo -e "\tdocker_port not defined in yaml"`)
		}
		if len(d.DockerVersion) == 0 {
			f.WriteCmdSilent(`echo -e "\tdocker_version not defined in yaml"`)
		}
		if len(d.ImageName) == 0 {
			f.WriteCmdSilent(`echo -e "\timage_name not defined in yaml"`)
		}
		return
	}

	f.WriteCmd("sudo apt-get update")

	// Ensure correct apt-get has the https method-driver as per (http://askubuntu.com/questions/165676/)
	f.WriteCmd("sudo apt-get install apt-transport-https")

	// Install Docker on the container
	f.WriteCmd("sudo sh -c \"echo deb https://get.docker.io/ubuntu docker main\\ > " +
		"/etc/apt/sources.list.d/docker.list\"")
	f.WriteCmd("sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys " +
		"36A1D7869245C8950F966E92D8576A8BA88D21E9")
	f.WriteCmd("sudo apt-get update")
	f.WriteCmd("sudo apt-get --yes install lxc-docker-" + d.DockerVersion)

	// Format our Build Server Endpoint
	dockerServerUrl := d.DockerServer + ":" + strconv.Itoa(d.DockerServerPort)

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
	f.WriteCmd(fmt.Sprintf("docker -H %s build -t %s:%s %s", dockerServerUrl, d.ImageName, imageTag, dockerPath))

	// Login?
	if d.RegistryLogin == true {
		// Are we logging in to a custom Registry?
		if len(d.RegistryLoginUrl) > 0 {
			f.WriteCmdSilent(fmt.Sprintf("docker -H %s login -u %s -p %s -e %s %s",
				dockerServerUrl, d.Username, d.Password, d.Email, d.RegistryLoginUrl))
		} else {
			// Assume index.docker.io
			f.WriteCmdSilent(fmt.Sprintf("docker -H %s login -u %s -p %s -e %s",
				dockerServerUrl, d.Username, d.Password, d.Email))
		}
	}

	// Are we overriding the "latest" tag?
	if d.PushLatest {
		f.WriteCmd(fmt.Sprintf("docker -H %s tag %s:%s %s:latest",
			dockerServerUrl, d.ImageName, imageTag, d.ImageName))
	}

	f.WriteCmd(fmt.Sprintf("docker -H %s push %s", dockerServerUrl, d.ImageName))

	// Delete the image from the docker server we built on.
	if !d.KeepBuild {
		f.WriteCmd(fmt.Sprintf("docker -H %s rmi %s:%s",
			dockerServerUrl, d.ImageName, imageTag))
		if d.PushLatest {
			f.WriteCmd(fmt.Sprintf("docker -H %s rmi %s:latest",
				dockerServerUrl, d.ImageName))
		}
	}
}
