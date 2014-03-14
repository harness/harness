package build

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/drone/drone/pkg/build/buildfile"
	"github.com/drone/drone/pkg/build/docker"
	"github.com/drone/drone/pkg/build/proxy"
	"github.com/drone/drone/pkg/build/repo"
	"github.com/drone/drone/pkg/build/script"
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

// Expected behavior is that a build script with no docker image
// specified will thrown an error.
func TestSetupEmptyImage(t *testing.T) {
	b := Builder{Build: &script.Build{}}
	var got, want = b.setup(), "Error: missing Docker image"

	if got == nil || got.Error() != want {
		t.Errorf("Expected error %s, got %s", want, got)
	}
}

// Expected behavior is that a build script with an unknown
// service (ie mysql)
func TestSetupUnknownService(t *testing.T) {
	b := Builder{}
	b.Repo = &repo.Repo{}
	b.Repo.Path = "git://github.com/drone/drone.git"
	b.Build = &script.Build{}
	b.Build.Image = "go1.2"
	b.Build.Services = append(b.Build.Services, "not-found")

	var got, want = b.setup(), "Error: Invalid or unknown service not-found"
	if got == nil || got.Error() != want {
		t.Errorf("Expected error %s, got %s", want, got)
	}
}

func TestSetupErrorRunDaemonPorts(t *testing.T) {
	setup()
	defer teardown()

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

func TestSetupErrorServiceInspect(t *testing.T) {
	setup()
	defer teardown()

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

func TestSetupErrorImagePull(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1.9/images/bradrydzewski/go:1.2/json", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	mux.HandleFunc("/v1.9/images/create", func(w http.ResponseWriter, r *http.Request) {
		// validate ?fromImage=bradrydzewski/go&tag=1.2
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

func TestTeardown(t *testing.T) {}

func TestTeardownContainerFail(t *testing.T) {}

func TestTeardownImageFail(t *testing.T) {}

func TestWriteIdentifyFile(t *testing.T) {
	// temporary directory to store file
	dir, _ := ioutil.TempDir("", "drone-test-")
	defer os.RemoveAll(dir)

	b := Builder{}
	b.Key = []byte("ssh-rsa AAA...")
	b.writeIdentifyFile(dir)

	// persist a dummy id_rsa keyfile to disk
	keyfile, err := ioutil.ReadFile(filepath.Join(dir, "id_rsa"))
	if err != nil {
		t.Errorf("Expected id_rsa file saved to disk")
	}

	if string(keyfile) != string(b.Key) {
		t.Errorf("Expected id_rsa value saved as %s, got %s", b.Key, keyfile)
	}
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
	f.WriteEnv("CI", "true")
	f.WriteEnv("DRONE", "true")
	f.WriteEnv("DRONE_BRANCH", "master")
	f.WriteEnv("DRONE_COMMIT", "e7e046b35")
	f.WriteEnv("DRONE_PR", "123")
	f.WriteEnv("DRONE_BUILD_DIR", "/var/cache/drone/github.com/drone/drone")
	f.WriteHost("127.0.0.1")
	f.WriteCmd("git clone --depth=0 --recursive --branch=master git://github.com/drone/drone.git /var/cache/drone/github.com/drone/drone")
	f.WriteCmd("git fetch origin +refs/pull/123/head:refs/remotes/origin/pr/123")
	f.WriteCmd("git checkout -qf -b pr/123 origin/pr/123")

	if string(script) != f.String() {
		t.Errorf("Expected build script value saved as %s, got %s", f.String(), script)
	}
}
