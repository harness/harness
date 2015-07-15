package main

import (
	"log"
	"time"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/citadel/citadel"
	"github.com/drone/drone/Godeps/_workspace/src/github.com/citadel/citadel/cluster"
	"github.com/drone/drone/Godeps/_workspace/src/github.com/citadel/citadel/scheduler"
)

type logHandler struct {
}

func (l *logHandler) Handle(e *citadel.Event) error {
	log.Printf("type: %s time: %s image: %s container: %s\n",
		e.Type, e.Time.Format(time.RubyDate), e.Container.Image.Name, e.Container.ID)

	return nil
}

func main() {
	boot2docker := &citadel.Engine{
		ID:     "boot2docker",
		Addr:   "http://192.168.56.102:2375",
		Memory: 2048,
		Cpus:   4,
		Labels: []string{"local"},
	}

	if err := boot2docker.Connect(nil); err != nil {
		log.Fatal(err)
	}

	c, err := cluster.New(scheduler.NewResourceManager(), boot2docker)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	if err := c.RegisterScheduler("service", &scheduler.LabelScheduler{}); err != nil {
		log.Fatal(err)
	}

	if err := c.Events(&logHandler{}); err != nil {
		log.Fatal(err)
	}

	image := &citadel.Image{
		Name:   "crosbymichael/redis",
		Memory: 256,
		Cpus:   0.4,
		Type:   "service",
	}

	container, err := c.Start(image, false)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("ran container %s\n", container.ID)

	containers := c.ListContainers(false, false, "")

	c1 := containers[0]

	if err := c.Kill(c1, 9); err != nil {
		log.Fatal(err)
	}

	if err := c.Remove(c1); err != nil {
		log.Fatal(err)
	}
}
