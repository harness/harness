package pool

import (
	"sync"

	"github.com/samalba/dockerclient"
)

// TODO (bradrydzewski) ability to cancel work.
// TODO (bradrydzewski) ability to remove a worker.

type Pool struct {
	sync.Mutex
	clients map[dockerclient.Client]bool
	clientc chan dockerclient.Client
}

func New() *Pool {
	return &Pool{
		clients: make(map[dockerclient.Client]bool),
		clientc: make(chan dockerclient.Client, 999),
	}
}

// Allocate allocates a client to the pool to
// be available to accept work.
func (p *Pool) Allocate(c dockerclient.Client) bool {
	if p.IsAllocated(c) {
		return false
	}

	p.Lock()
	p.clients[c] = true
	p.Unlock()

	p.clientc <- c
	return true
}

// IsAllocated is a helper function that returns
// true if the client is currently allocated to
// the Pool.
func (p *Pool) IsAllocated(c dockerclient.Client) bool {
	p.Lock()
	defer p.Unlock()
	_, ok := p.clients[c]
	return ok
}

// Deallocate removes the worker from the pool of
// available clients. If the client is currently
// reserved and performing work it will finish,
// but no longer be given new work.
func (p *Pool) Deallocate(c dockerclient.Client) {
	p.Lock()
	defer p.Unlock()
	delete(p.clients, c)
}

// List returns a list of all Workers currently
// allocated to the Pool.
func (p *Pool) List() []dockerclient.Client {
	p.Lock()
	defer p.Unlock()

	var clients []dockerclient.Client
	for c := range p.clients {
		clients = append(clients, c)
	}
	return clients
}

// Reserve reserves the next available worker to
// start doing work. Once work is complete, the
// worker should be released back to the pool.
func (p *Pool) Reserve() <-chan dockerclient.Client {
	return p.clientc
}

// Release releases the worker back to the pool
// of available workers.
func (p *Pool) Release(c dockerclient.Client) bool {
	if !p.IsAllocated(c) {
		return false
	}

	p.clientc <- c
	return true
}
