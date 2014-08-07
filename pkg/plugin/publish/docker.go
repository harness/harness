package publish

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/drone/drone/pkg/build/buildfile"
	"github.com/drone/drone/pkg/build/repo"
)

type Docker struct {
	// The path to the dockerfile to create the image from. If the path is empty or no
	// path is specified then the docker file will be built from the base directory.
	Dockerfile string `yaml:"docker_file"`

	// Connection information for the docker server that will build the image
	DockerServer string `yaml:"docker_server"`
	DockerServerPort   int    `yaml:"docker_port"`
	// The Docker client version to download. This must match the docker version on the server
	DockerVersion string `yaml:"docker_version"`

	// Optional Arguments to allow finer-grained control of registry
	// endpoints
	RegistryHost string `yaml:"registry_host"`
	RegistryProtocol string `yaml:"registry_protocol"`
	RegistryPort int `yaml:"registry_port"`
    RegistryLogin bool `yaml:"registry_login"`
	RegistryLoginUri string `yaml:"registry_login_uri"`

	ImageName string `yaml:"image_name"`

	// Authentication credentials for index.docker.io
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Email    string `yaml:"email"`

	// Keep the build on the Docker host after pushing?
	KeepBuild bool `yaml:"keep_build"`
    // Do we want to override "latest" automatically with this build?
    PushLatest bool `yaml:"push_latest"`

	Branch string `yaml:"branch,omitempty"`
	Tag string `yaml:"custom_tag"`
}

// Write adds commands to the buildfile to do the following:
// 1. Install the docker client in the Drone container.
// 2. Build a docker image based on the dockerfile defined in the config.
// 3. Push that docker image to index.docker.io.
// 4. Delete the docker image on the server it was build on so we conserve disk space.
func (d *Docker) Write(f *buildfile.Buildfile, r *repo.Repo) {
	if len(d.DockerServer) == 0 || d.DockerServerPort == 0 || len(d.DockerVersion) == 0 ||
		len(d.ImageName) == 0 {
		f.WriteCmdSilent(`echo "Docker Plugin: Missing argument(s)"`)
		return
	}

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

	// Construct Image BaseName 
	// e.g. "docker.mycompany.com/myimage" for private registries
	//      "myuser/myimage" for index.docker.io
	imageBaseName := ""
	if len(d.RegistryHost) > 0 {
		imageBaseName = fmt.Sprintf("%s/%s",d.RegistryHost,d.ImageName)
	} else {
		imageBaseName = fmt.Sprintf("%s/%s",d.Username,d.ImageName)
	}

	registryLoginEndpoint := ""

	// Gather information to build our Registry Endpoint for private registries
	if len(d.RegistryHost) > 0 {
		// Set Protocol
		if len(d.RegistryProtocol) > 0 {
			registryLoginEndpoint = fmt.Sprintf("%s://%s", d.RegistryProtocol,d.RegistryHost)
		} else {
			registryLoginEndpoint = fmt.Sprintf("http://%s", d.RegistryHost)
		}
		// Set Port
		if d.RegistryPort > 0 {
			registryLoginEndpoint = fmt.Sprintf("%s:%d",registryLoginEndpoint,d.RegistryPort)
		}
		// Set Login URI
		if len(d.RegistryLoginUri) > 0 {
			registryLoginEndpoint = fmt.Sprintf("%s/%s",registryLoginEndpoint,strings.TrimPrefix(d.RegistryLoginUri,"/"))
		} else {
			registryLoginEndpoint = fmt.Sprintf("%s/v1/",registryLoginEndpoint)
		}
	}

	//splitRepoName := strings.Split(r.Name, "/")
	//dockerRepo := d.ImageName + "/" + splitRepoName[len(splitRepoName)-1]

	dockerPath := "."
	if len(d.Dockerfile) != 0 {
		dockerPath = fmt.Sprintf("- < %s", d.Dockerfile)
	}

	// Run the command commands to build and deploy the image.
	// Are we setting a custom tag, or do we use the git hash?
	imageTag := ""
	if len(d.Tag) > 0 {
		imageTag = d.Tag
	} else {
		imageTag = "$(git rev-parse --short HEAD)"
	}
	f.WriteCmd(fmt.Sprintf("docker -H %s build -t %s:%s %s", dockerServerUrl, imageBaseName, imageTag, dockerPath))

	// Login?
	if len(d.RegistryHost) > 0 && d.RegistryLogin == true {
		f.WriteCmdSilent(fmt.Sprintf("docker -H %s login -u %s -p %s -e %s %s",
			dockerServerUrl, d.Username, d.Password, d.Email, registryLoginEndpoint))
	} else if len(d.RegistryHost) == 0 {
		// Assume that because no private registry is specified it requires auth
		// for index.docker.io
		f.WriteCmdSilent(fmt.Sprintf("docker -H %s login -u %s -p %s -e %s",
			dockerServerUrl, d.Username, d.Password, d.Email))
	}

    // Are we overriding the "latest" tag?
    if d.PushLatest {
		f.WriteCmd(fmt.Sprintf("docker -H %s tag %s:%s %s:latest",
			dockerServerUrl, imageBaseName, imageTag, imageBaseName))
    }

	f.WriteCmd(fmt.Sprintf("docker -H %s push %s", dockerServerUrl, imageBaseName))

	// Delete the image from the docker server we built on.
	if ! d.KeepBuild {
		f.WriteCmd(fmt.Sprintf("docker -H %s rmi %s:%s",
			dockerServerUrl, imageBaseName, imageTag))
		f.WriteCmd(fmt.Sprintf("docker -H %s rmi %s:latest",
			dockerServerUrl, imageBaseName))
	}
}
