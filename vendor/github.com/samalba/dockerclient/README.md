Docker client library in Go
===========================
[![GoDoc](http://godoc.org/github.com/samalba/dockerclient?status.png)](http://godoc.org/github.com/samalba/dockerclient)

Well maintained docker client library.

# How to use it?

Here is an example showing how to use it:

```go
package main

import (
	"github.com/samalba/dockerclient"
	"log"
	"time"
	"os"
)

// Callback used to listen to Docker's events
func eventCallback(event *dockerclient.Event, ec chan error, args ...interface{}) {
	log.Printf("Received event: %#v\n", *event)
}

func main() {
	// Init the client
	docker, _ := dockerclient.NewDockerClient("unix:///var/run/docker.sock", nil)

	// Get only running containers
	containers, err := docker.ListContainers(false, false, "")
	if err != nil {
		log.Fatal(err)
	}
	for _, c := range containers {
		log.Println(c.Id, c.Names)
	}

	// Inspect the first container returned
	if len(containers) > 0 {
		id := containers[0].Id
		info, _ := docker.InspectContainer(id)
		log.Println(info)
	}

	// Build a docker image
	// some.tar contains the build context (Dockerfile any any files it needs to add/copy)
	dockerBuildContext, err := os.Open("some.tar")
	defer dockerBuildContext.Close()
	buildImageConfig := &dockerclient.BuildImage{
			Context:        dockerBuildContext,
			RepoName:       "your_image_name",
			SuppressOutput: false,
	}
	reader, err := docker.BuildImage(buildImageConfig)
	if err != nil {
		log.Fatal(err)
	}

	// Create a container
	containerConfig := &dockerclient.ContainerConfig{
		Image: "ubuntu:14.04",
		Cmd:   []string{"bash"},
		AttachStdin: true,
		Tty:   true}
	containerId, err := docker.CreateContainer(containerConfig, "foobar", nil)
	if err != nil {
		log.Fatal(err)
	}

	// Start the container
	hostConfig := &dockerclient.HostConfig{}
	err = docker.StartContainer(containerId, hostConfig)
	if err != nil {
		log.Fatal(err)
	}

	// Stop the container (with 5 seconds timeout)
	docker.StopContainer(containerId, 5)

	// Listen to events
	docker.StartMonitorEvents(eventCallback, nil)

	// Hold the execution to look at the events coming
	time.Sleep(3600 * time.Second)
}
```

# Maintainers

List of people you can ping for feedback on Pull Requests or any questions.

- [Sam Alba](https://github.com/samalba)
- [Michael Crosby](https://github.com/crosbymichael)
- [Andrea Luzzardi](https://github.com/aluzzardi)
- [Victor Vieux](https://github.com/vieux)
- [Evan Hazlett](https://github.com/ehazlett)
- [Donald Huang](https://github.com/donhcd)
