package main

import (
	"github.com/samalba/dockerclient"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func statCallback(id string, stat *dockerclient.Stats, ec chan error, args ...interface{}) {
	log.Println(stat)
}

func waitForInterrupt() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	for _ = range sigChan {
		os.Exit(0)
	}
}

func main() {
	docker, err := dockerclient.NewDockerClient(os.Getenv("DOCKER_HOST"), nil)
	if err != nil {
		log.Fatal(err)
	}

	containerConfig := &dockerclient.ContainerConfig{Image: "busybox", Cmd: []string{"sh"}}
	containerId, err := docker.CreateContainer(containerConfig, "", nil)
	if err != nil {
		log.Fatal(err)
	}

	// Start the container
	err = docker.StartContainer(containerId, nil)
	if err != nil {
		log.Fatal(err)
	}
	docker.StartMonitorStats(containerId, statCallback, nil)

	waitForInterrupt()
}
