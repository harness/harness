package cluster

import (
	"sync"

	"github.com/samalba/dockerclient"
)

// TODO (bradrydzewski) ability to cancel work.
// TODO (bradrydzewski) ability to remove a worker.

type Cluster struct {
	sync.Mutex
	clients map[dockerclient.Client]bool
	clientc chan dockerclient.Client
}

func New() *Cluster {
	return &Cluster{
		clients: make(map[dockerclient.Client]bool),
		clientc: make(chan dockerclient.Client, 999),
	}
}

// Allocate allocates a client to the pool to
// be available to accept work.
func (c *Cluster) Allocate(cli dockerclient.Client) bool {
	if c.IsAllocated(cli) {
		return false
	}

	c.Lock()
	c.clients[cli] = true
	c.Unlock()

	c.clientc <- cli
	return true
}

// IsAllocated is a helper function that returns
// true if the client is currently allocated to
// the Pool.
func (c *Cluster) IsAllocated(cli dockerclient.Client) bool {
	c.Lock()
	defer c.Unlock()
	_, ok := c.clients[cli]
	return ok
}

// Deallocate removes the worker from the pool of
// available clients. If the client is currently
// reserved and performing work it will finish,
// but no longer be given new work.
func (c *Cluster) Deallocate(cli dockerclient.Client) {
	c.Lock()
	defer c.Unlock()
	delete(c.clients, cli)
}

// List returns a list of all Workers currently
// allocated to the Pool.
func (c *Cluster) List() []dockerclient.Client {
	c.Lock()
	defer c.Unlock()

	var clients []dockerclient.Client
	for cli := range c.clients {
		clients = append(clients, cli)
	}
	return clients
}

// Reserve reserves the next available worker to
// start doing work. Once work is complete, the
// worker should be released back to the pool.
func (p *Cluster) Reserve() <-chan dockerclient.Client {
	return p.clientc
}

// Release releases the worker back to the pool
// of available workers.
func (c *Cluster) Release(cli dockerclient.Client) bool {
	if !c.IsAllocated(cli) {
		return false
	}

	c.clientc <- cli
	return true
}
