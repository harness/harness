package publish

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/drone/drone/pkg/build/buildfile"
	"github.com/drone/drone/pkg/build/repo"
)

type Docker struct {
	// The path to the dockerfile to create the image from
	Dockerfile string `yaml:"docker_file"`

	// Connection information for the docker server that will build the image
	Server string `yaml:"docker_server"`
	Port   int    `yaml:"docker_port"`
	// The Docker client version to download. This must match the docker version on the server
	DockerVersion string `yaml:"docker_version"`

	RepoBaseName string `yaml:"repo_base_name"`

	// Authentication credentials for index.docker.io
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Email    string `yaml:"email"`

	Branch string `yaml:"branch,omitempty"`
}

// Write adds commands to the buildfile to do the following:
// 1. Install the docker client in the Drone container.
// 2. Build a docker image based on the dockerfile defined in the config.
// 3. Push that docker image to index.docker.io.
// 4. Delete the docker image on the server it was build on so we conserve disk space.
func (d *Docker) Write(f *buildfile.Buildfile, r *repo.Repo) {
	if len(d.Dockerfile) == 0 || len(d.Server) == 0 || d.Port == 0 || len(d.DockerVersion) == 0 ||
		len(d.RepoBaseName) == 0 || len(d.Username) == 0 || len(d.Password) == 0 ||
		len(d.Email) == 0 {
		f.WriteCmdSilent(`echo "Docker Plugin: Missing argument(s)"`)
		return
	}

	// Install Docker on the container
	f.WriteCmd("sudo sh -c \"echo deb http://get.docker.io/ubuntu docker main\\ > " +
		"/etc/apt/sources.list.d/docker.list\"")
	f.WriteCmd("sudo apt-get update")
	f.WriteCmd("sudo apt-get --yes install lxc-docker-" + d.DockerVersion)

	dockerServerUrl := d.Server + ":" + strconv.Itoa(d.Port)
	splitRepoName := strings.Split(r.Name, "/")
	dockerRepo := d.RepoBaseName + "/" + splitRepoName[len(splitRepoName) - 1]
	// Run the command commands to build and deploy the image. Note that the image is tagged
	// with the git hash.
	f.WriteCmd(fmt.Sprintf("docker -H %s build -t %s:$(git rev-parse --short HEAD) - < %s",
		dockerServerUrl, dockerRepo, d.Dockerfile))

	// Login and push to index.docker.io
	f.WriteCmdSilent(fmt.Sprintf("docker -H %s login -u %s -p %s -e %s",
		dockerServerUrl, d.Username, d.Password, d.Email))
	f.WriteCmd(fmt.Sprintf("docker -H %s push %s", dockerServerUrl, dockerRepo))

        // Delete the image from the docker server we built on.
	f.WriteCmd(fmt.Sprintf("docker -H %s rmi %s:$(git rev-parse --short HEAD)",
		dockerServerUrl, dockerRepo))
}
