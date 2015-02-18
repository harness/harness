package build

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/drone/drone/shared/build/buildfile"
	"github.com/drone/drone/shared/build/docker"
	"github.com/drone/drone/shared/build/proxy"
	"github.com/drone/drone/shared/build/repo"
	"github.com/drone/drone/shared/build/script"
)

var (
	// mux is the HTTP request multiplexer used with the test server.
	mux *http.ServeMux

	// server is a test HTTP server used to provide mock API responses.
	server *httptest.Server

	// docker client
	client *docker.Client
)

// setup a mock docker client for testing purposes. This will use
// a test http server that can return mock responses to the docker client.
func setup() {
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)

	url, _ := url.Parse(server.URL)
	url.Scheme = "tcp"
	os.Setenv("DOCKER_HOST", url.String())
	client = docker.New()
}

func teardown() {
	server.Close()
}

// TestSetup will test our ability to successfully create a Docker
// image for the build.
func TestSetup(t *testing.T) {
	setup()
	defer teardown()

	// Handles a request to inspect the Go 1.2 image
	// This will return a dummy image ID, so that the system knows
	// the build image exists, and doens't need to be downloaded.
	mux.HandleFunc("/v1.9/images/bradrydzewski/go:1.2/json", func(w http.ResponseWriter, r *http.Request) {
		body := `[{ "id": "7bf9ce0ffb7236ca68da0f9fed0e1682053b393db3c724ff3c5a4e8c0793b34c" }]`
		w.Write([]byte(body))
	})

	// Handles a request to create the build image, with the build
	// script injected. This will return a dummy stream.
	mux.HandleFunc("/v1.9/build", func(w http.ResponseWriter, r *http.Request) {
		body := `{"stream":"Step 1..."}`
		w.Write([]byte(body))
	})

	// Handles a request to inspect the newly created build image. Note
	// that we are doing a "wildcard" url match here, since the name of
	// the image will be random. This will return a dummy image ID
	// to confirm the build image was created successfully.
	mux.HandleFunc("/v1.9/images/", func(w http.ResponseWriter, r *http.Request) {
		body := `{ "id": "7bf9ce0ffb7236ca68da0f9fed0e1682053b393db3c724ff3c5a4e8c0793b34c" }`
		w.Write([]byte(body))
	})

	b := Builder{}
	b.Repo = &repo.Repo{}
	b.Repo.Path = "git://github.com/drone/drone.git"
	b.Build = &script.Build{}
	b.Build.Image = "go1.2"
	b.dockerClient = client

	if err := b.setup(); err != nil {
		t.Errorf("Expected success, got %s", err)
	}

	// verify the Image is being correctly set
	if b.image == nil {
		t.Errorf("Expected image not nil")
	}

	expectedID := "7bf9ce0ffb7236ca68da0f9fed0e1682053b393db3c724ff3c5a4e8c0793b34c"
	if b.image.ID != expectedID {
		t.Errorf("Expected image.ID %s, got %s", expectedID, b.image.ID)
	}
}

// TestSetupEmptyImage will test our ability to handle a nil or
// blank Docker build image. We expect this to return an error.
func TestSetupEmptyImage(t *testing.T) {
	b := Builder{Build: &script.Build{}}
	var got, want = b.setup(), "Error: missing Docker image"

	if got == nil || got.Error() != want {
		t.Errorf("Expected error %s, got %s", want, got)
	}
}

// TestSetupErrorInspectImage will test our ability to handle a
// failure when inspecting an image (i.e. bradrydzewski/mysql:latest),
// which should trigger a `docker pull`.
func TestSetupErrorInspectImage(t *testing.T) {
	t.Skip()
}

// TestSetupErrorPullImage will test our ability to handle a
// failure when pulling an image (i.e. bradrydzewski/mysql:latest)
func TestSetupErrorPullImage(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1.9/images/bradrydzewski/mysql:5.5/json", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

}

// TestSetupErrorRunDaemonPorts will test our ability to handle a
// failure when starting a service (i.e. mysql) as a daemon.
func TestSetupErrorRunDaemonPorts(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1.9/images/bradrydzewski/mysql:5.5/json", func(w http.ResponseWriter, r *http.Request) {
		data := []byte(`{"config": { "ExposedPorts": { "6379/tcp": {}}}}`)
		w.Write(data)
	})

	mux.HandleFunc("/v1.9/containers/create", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	b := Builder{}
	b.Repo = &repo.Repo{}
	b.Repo.Path = "git://github.com/drone/drone.git"
	b.Build = &script.Build{}
	b.Build.Image = "go1.2"
	b.Build.Services = append(b.Build.Services, "mysql")
	b.dockerClient = client

	var got, want = b.setup(), docker.ErrBadRequest
	if got == nil || got != want {
		t.Errorf("Expected error %s, got %s", want, got)
	}
}

// TestSetupErrorServiceInspect will test our ability to handle a
// failure when a service (i.e. mysql) is started successfully,
// but cannot be queried post-start with the Docker remote API.
func TestSetupErrorServiceInspect(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1.9/images/bradrydzewski/mysql:5.5/json", func(w http.ResponseWriter, r *http.Request) {
		data := []byte(`{"config": { "ExposedPorts": { "6379/tcp": {}}}}`)
		w.Write(data)
	})

	mux.HandleFunc("/v1.9/containers/create", func(w http.ResponseWriter, r *http.Request) {
		body := `{ "Id":"e90e34656806", "Warnings":[] }`
		w.Write([]byte(body))
	})

	mux.HandleFunc("/v1.9/containers/e90e34656806/start", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("/v1.9/containers/e90e34656806/json", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	b := Builder{}
	b.Repo = &repo.Repo{}
	b.Repo.Path = "git://github.com/drone/drone.git"
	b.Build = &script.Build{}
	b.Build.Image = "go1.2"
	b.Build.Services = append(b.Build.Services, "mysql")
	b.dockerClient = client

	var got, want = b.setup(), docker.ErrBadRequest
	if got == nil || got != want {
		t.Errorf("Expected error %s, got %s", want, got)
	}
}

// TestSetupErrorImagePull will test our ability to handle a
// failure when a the build image cannot be pulled from the index.
func TestSetupErrorImagePull(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1.9/images/bradrydzewski/mysql:5.5/json", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	mux.HandleFunc("/v1.9/images/create?fromImage=bradrydzewski/mysql&tag=5.5", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	b := Builder{}
	b.Repo = &repo.Repo{}
	b.Repo.Path = "git://github.com/drone/drone.git"
	b.Build = &script.Build{}
	b.Build.Image = "go1.2"
	b.Build.Services = append(b.Build.Services, "mysql")
	b.dockerClient = client

	var got, want = b.setup(), fmt.Errorf("Error: Unable to pull image bradrydzewski/mysql:5.5")
	if got == nil || got.Error() != want.Error() {
		t.Errorf("Expected error %s, got %s", want, got)
	}
}

// TestSetupErrorBuild will test our ability to handle a failure
// when creating a Docker image with the injected build script,
// ssh keys, etc.
func TestSetupErrorBuild(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1.9/images/bradrydzewski/go:1.2/json", func(w http.ResponseWriter, r *http.Request) {
		body := `[{ "id": "7bf9ce0ffb7236ca68da0f9fed0e1682053b393db3c724ff3c5a4e8c0793b34c" }]`
		w.Write([]byte(body))
	})

	mux.HandleFunc("/v1.9/build", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	b := Builder{}
	b.Repo = &repo.Repo{}
	b.Repo.Path = "git://github.com/drone/drone.git"
	b.Build = &script.Build{}
	b.Build.Image = "go1.2"
	b.dockerClient = client

	var got, want = b.setup(), docker.ErrBadRequest
	if got == nil || got != want {
		t.Errorf("Expected error %s, got %s", want, got)
	}
}

// TestSetupErrorBuildInspect will test our ability to handle a failure
// when we successfully create a Docker image with the injected build script,
// ssh keys, etc, however, we cannot inspect it post-creation using
// the Docker remote API.
func TestSetupErrorBuildInspect(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1.9/images/bradrydzewski/go:1.2/json", func(w http.ResponseWriter, r *http.Request) {
		body := `[{ "id": "7bf9ce0ffb7236ca68da0f9fed0e1682053b393db3c724ff3c5a4e8c0793b34c" }]`
		w.Write([]byte(body))
	})

	mux.HandleFunc("/v1.9/build", func(w http.ResponseWriter, r *http.Request) {
		body := `{"stream":"Step 1..."}`
		w.Write([]byte(body))
	})

	mux.HandleFunc("/v1.9/images/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	b := Builder{}
	b.Repo = &repo.Repo{}
	b.Repo.Path = "git://github.com/drone/drone.git"
	b.Build = &script.Build{}
	b.Build.Image = "go1.2"
	b.dockerClient = client

	var got, want = b.setup(), docker.ErrBadRequest
	if got == nil || got != want {
		t.Errorf("Expected error %s, got %s", want, got)
	}
}

// TestTeardown will test our ability to sucessfully teardown a
// Docker-based build environment.
func TestTeardown(t *testing.T) {
	setup()
	defer teardown()

	var (
		containerStopped = false
		containerRemoved = false
		serviceStopped   = false
		serviceRemoved   = false
		imageRemoved     = false
	)

	mux.HandleFunc("/v1.9/containers/7bf9ce0ffb/stop", func(w http.ResponseWriter, r *http.Request) {
		containerStopped = true
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/v1.9/containers/7bf9ce0ffb", func(w http.ResponseWriter, r *http.Request) {
		containerRemoved = true
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/v1.9/containers/ec62dcc736/stop", func(w http.ResponseWriter, r *http.Request) {
		serviceStopped = true
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/v1.9/containers/ec62dcc736", func(w http.ResponseWriter, r *http.Request) {
		serviceRemoved = true
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/v1.9/images/c3ab8ff137", func(w http.ResponseWriter, r *http.Request) {
		imageRemoved = true
		w.Write([]byte(`[{"Untagged":"c3ab8ff137"},{"Deleted":"c3ab8ff137"}]`))
	})

	b := Builder{}
	b.dockerClient = client
	b.services = append(b.services, &docker.Container{ID: "ec62dcc736"})
	b.container = &docker.Run{ID: "7bf9ce0ffb"}
	b.image = &docker.Image{ID: "c3ab8ff137"}
	b.Build = &script.Build{Services: []string{"mysql"}}
	b.teardown()

	if !containerStopped {
		t.Errorf("Expected Docker container was stopped")
	}

	if !containerRemoved {
		t.Errorf("Expected Docker container was removed")
	}

	if !serviceStopped {
		t.Errorf("Expected Docker mysql container was stopped")
	}

	if !serviceRemoved {
		t.Errorf("Expected Docker mysql container was removed")
	}

	if !imageRemoved {
		t.Errorf("Expected Docker image was removed")
	}
}

func TestRun(t *testing.T) {
	t.Skip()
}

func TestRunPrivileged(t *testing.T) {
	setup()
	defer teardown()

	var conf = docker.HostConfig{}

	mux.HandleFunc("/v1.9/containers/create", func(w http.ResponseWriter, r *http.Request) {
		body := `{ "Id":"e90e34656806", "Warnings":[] }`
		w.Write([]byte(body))
	})

	mux.HandleFunc("/v1.9/containers/e90e34656806/start", func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&conf)
		w.WriteHeader(http.StatusBadRequest)
	})

	b := Builder{}
	b.BuildState = &BuildState{}
	b.dockerClient = client
	b.Stdout = new(bytes.Buffer)
	b.image = &docker.Image{ID: "c3ab8ff137"}
	b.Build = &script.Build{}
	b.Repo = &repo.Repo{}
	b.run()

	if conf.Privileged != false {
		t.Errorf("Expected container NOT started in Privileged mode")
	}

	// now lets set priviliged mode
	b.Privileged = true
	b.run()

	if conf.Privileged != true {
		t.Errorf("Expected container IS started in Privileged mode")
	}

	// now lets set priviliged mode but for a pull request
	b.Privileged = true
	b.Repo.PR = "55"
	b.run()

	if conf.Privileged != false {
		t.Errorf("Expected container NOT started in Privileged mode when PR")
	}
}

func TestRunErrorCreate(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1.9/containers/create", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	b := Builder{}
	b.BuildState = &BuildState{}
	b.dockerClient = client
	b.Stdout = new(bytes.Buffer)
	b.image = &docker.Image{ID: "c3ab8ff137"}
	b.Build = &script.Build{}
	b.Repo = &repo.Repo{}
	if err := b.run(); err == nil || err.Error() != "Error: Failed to create build container. Bad Request" {
		t.Errorf("Expected error when trying to create build container")
	}
}

func TestRunErrorStart(t *testing.T) {
	setup()
	defer teardown()

	var (
		containerCreated = false
		containerStarted = false
	)

	mux.HandleFunc("/v1.9/containers/create", func(w http.ResponseWriter, r *http.Request) {
		containerCreated = true
		body := `{ "Id":"e90e34656806", "Warnings":[] }`
		w.Write([]byte(body))
	})

	mux.HandleFunc("/v1.9/containers/e90e34656806/start", func(w http.ResponseWriter, r *http.Request) {
		containerStarted = true
		w.WriteHeader(http.StatusBadRequest)
	})

	b := Builder{}
	b.BuildState = &BuildState{}
	b.dockerClient = client
	b.Stdout = new(bytes.Buffer)
	b.image = &docker.Image{ID: "c3ab8ff137"}
	b.Build = &script.Build{}
	b.Repo = &repo.Repo{}

	if err := b.run(); err == nil || err.Error() != "Error: Failed to start build container. Bad Request" {
		t.Errorf("Expected error when trying to start build container")
	}

	if !containerCreated {
		t.Errorf("Expected Docker endpoint was invoked to create container")
	}

	if !containerStarted {
		t.Errorf("Expected Docker endpoint was invoked to start container")
	}

	if b.container == nil || b.container.ID != "e90e34656806" {
		t.Errorf("Expected build container was created with ID e90e34656806")
	}
}

func TestRunErrorWait(t *testing.T) {
	t.Skip()
}

func TestWriteProxyScript(t *testing.T) {
	// temporary directory to store file
	dir, _ := ioutil.TempDir("", "drone-test-")
	defer os.RemoveAll(dir)

	// fake service container that we'll assume was part of the yaml
	// and should be attached to the build container.
	c := docker.Container{
		NetworkSettings: &docker.NetworkSettings{
			IPAddress: "172.1.4.5",
			Ports: map[docker.Port][]docker.PortBinding{
				docker.NewPort("tcp", "3306"): nil,
			},
		},
	}

	// this should generate the following proxy file
	p := proxy.Proxy{}
	p.Set("3306", "172.1.4.5")
	want := p.String()

	b := Builder{}
	b.services = append(b.services, &c)
	b.writeProxyScript(dir)

	// persist a dummy proxy script to disk
	got, err := ioutil.ReadFile(filepath.Join(dir, "proxy.sh"))
	if err != nil {
		t.Errorf("Expected proxy.sh file saved to disk")
	}

	if string(got) != want {
		t.Errorf("Expected proxy.sh value saved as %s, got %s", want, got)
	}
}

func TestWriteBuildScript(t *testing.T) {
	// temporary directory to store file
	dir, _ := ioutil.TempDir("", "drone-test-")
	defer os.RemoveAll(dir)

	b := Builder{}
	b.Build = &script.Build{
		Hosts: []string{"127.0.0.1"}}
	b.Key = []byte("ssh-rsa AAA...")
	b.Repo = &repo.Repo{
		Path:   "git://github.com/drone/drone.git",
		Branch: "master",
		Commit: "e7e046b35",
		PR:     "123",
		Dir:    "/var/cache/drone/github.com/drone/drone"}
	b.writeBuildScript(dir)

	// persist a dummy build script to disk
	script, err := ioutil.ReadFile(filepath.Join(dir, "drone"))
	if err != nil {
		t.Errorf("Expected id_rsa file saved to disk")
	}

	f := buildfile.New()
	f.WriteEnv("TERM", "xterm")
	f.WriteEnv("GOPATH", "/var/cache/drone")
	f.WriteEnv("SHELL", "/bin/bash")
	f.WriteEnv("CI", "true")
	f.WriteEnv("DRONE", "true")
	f.WriteEnv("DRONE_REMOTE", "git://github.com/drone/drone.git")
	f.WriteEnv("DRONE_BRANCH", "master")
	f.WriteEnv("DRONE_COMMIT", "e7e046b35")
	f.WriteEnv("DRONE_PR", "123")
	f.WriteEnv("DRONE_BUILD_DIR", "/var/cache/drone/github.com/drone/drone")
	f.WriteEnv("CI_NAME", "DRONE")
	f.WriteEnv("CI_BUILD_URL", "")
	f.WriteEnv("CI_REMOTE", "git://github.com/drone/drone.git")
	f.WriteEnv("CI_BRANCH", "master")
	f.WriteEnv("CI_PULL_REQUEST", "123")
	f.WriteHost("127.0.0.1")
	f.WriteFile("$HOME/.ssh/id_rsa", []byte("ssh-rsa AAA..."), 600)
	f.WriteCmd("git clone --depth=0 --recursive git://github.com/drone/drone.git /var/cache/drone/github.com/drone/drone")
	f.WriteCmd("git fetch origin +refs/pull/123/head:refs/remotes/origin/pr/123")
	f.WriteCmd("git checkout -qf -b pr/123 origin/pr/123")

	if string(script) != f.String() {
		t.Errorf("Expected build script value saved as %s, got %s", f.String(), script)
	}
}
