package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

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
				time.Sleep(30 * time.Second)
				continue
			}
			runner_ := runner.Runner{&updater{addr, token}}
			runner_.Run(w)
		}
	}()

	s := gin.New()
	s.GET("/stream/:id", stream)
	s.GET("/ping", ping)
	s.GET("/about", about)
	s.Run(":1999")
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

func pull() (*queue.Work, error) {
	url_, _ := url.Parse(addr)
	url_.Path = "/api/queue/pull"
	var body bytes.Buffer
	resp, err := http.Post(url_.String(), "application/json", &body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	work := &queue.Work{}
	err = json.NewDecoder(resp.Body).Decode(work)
	return work, err
}
