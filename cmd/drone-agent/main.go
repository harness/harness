package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/drone/drone/pkg/queue"
	runner "github.com/drone/drone/pkg/runner/builtin"

	"github.com/gin-gonic/gin"
	"github.com/samalba/dockerclient"
)

var (
	// commit sha for the current build, set by
	// the compile process.
	version  string
	revision string
)

var (
	// Defult docker host address
	DefaultHost = "unix:///var/run/docker.sock"

	// Docker host address from environment variable
	DockerHost = os.Getenv("DOCKER_HOST")
)

var (
	addr  string
	token string
)

func main() {
	flag.StringVar(&addr, "addr", "http://localhost:8080", "")
	flag.StringVar(&token, "token", "", "")
	flag.Parse()

	if len(DockerHost) == 0 {
		DockerHost = DefaultHost
	}

	go func() {
		for {
			w, err := pull()
			if err != nil {
				log.Errorln(err)
				time.Sleep(30 * time.Second)
				continue
			}

			log.Infof("Pulled and running build %s / %d",
				w.Repo.FullName, w.Commit.Sequence)

			updater := &updater{}
			runner_ := runner.Runner{Updater: updater}
			err = runner_.Run(w)
			if err != nil {
				log.Errorln(err)
			}
		}
	}()

	s := gin.New()
	s.GET("/stream/:id", stream)
	s.GET("/ping", ping)
	s.GET("/about", about)
	s.Run(":1999")
}

func pull() (*queue.Work, error) {
	out := &queue.Work{}
	err := send("POST", "/api/queue/pull", nil, out)
	return out, err
}

// ping handler returns a simple response to the
// caller indicating the server is running. This
// can be used for heartbeats.
func ping(c *gin.Context) {
	c.String(200, "PONG")
}

// about handler returns the version and revision
// information for this server.
func about(c *gin.Context) {
	out := struct {
		Version  string
		Revision string
	}{version, revision}
	c.JSON(200, out)
}

// stream handler is a proxy that streams the Docker
// stdout and stderr for a running build to the caller.
func stream(c *gin.Context) {
	if c.Request.FormValue("token") != token {
		c.AbortWithStatus(401)
		return
	}

	client, err := dockerclient.NewDockerClient(DockerHost, nil)
	if err != nil {
		c.Fail(500, err)
		return
	}
	cname := fmt.Sprintf("drone-%s", c.Params.ByName("id"))

	// finds the container by name
	info, err := client.InspectContainer(cname)
	if err != nil {
		c.Fail(404, err)
		return
	}

	// verify the container is running. if not we'll
	// do an exponential backoff and attempt to wait
	if !info.State.Running {
		for i := 0; ; i++ {
			time.Sleep(1 * time.Second)
			info, err = client.InspectContainer(info.Id)
			if err != nil {
				c.Fail(404, err)
				return
			}
			if info.State.Running {
				break
			}
			if i == 5 {
				c.Fail(404, dockerclient.ErrNotFound)
				return
			}
		}
	}

	logs := &dockerclient.LogOptions{
		Follow: true,
		Stdout: true,
		Stderr: true,
	}

	// directly streams the build output from the Docker
	// daemon to the request.
	rc, err := client.ContainerLogs(info.Id, logs)
	if err != nil {
		c.Fail(500, err)
		return
	}
	io.Copy(c.Writer, rc)
}
