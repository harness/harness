package pool

import (
	"sync"

	"github.com/drone/drone/server/worker"
)

// TODO (bradrydzewski) ability to cancel work.
// TODO (bradrydzewski) ability to remove a worker.

type Pool struct {
	sync.Mutex
	workers map[worker.Worker]bool
	workerc chan worker.Worker
}

func New() *Pool {
	return &Pool{
		workers: make(map[worker.Worker]bool),
		workerc: make(chan worker.Worker, 999),
	}
}

// Allocate allocates a worker to the pool to
// be available to accept work.
func (p *Pool) Allocate(w worker.Worker) bool {
	if p.IsAllocated(w) {
		return false
	}

	p.Lock()
	p.workers[w] = true
	p.Unlock()

	p.workerc <- w
	return true
}

// IsAllocated is a helper function that returns
// true if the worker is currently allocated to
// the Pool.
func (p *Pool) IsAllocated(w worker.Worker) bool {
	p.Lock()
	defer p.Unlock()
	_, ok := p.workers[w]
	return ok
}

// Deallocate removes the worker from the pool of
// available workers. If the worker is currently
// reserved and performing work it will finish,
// but no longer be given new work.
func (p *Pool) Deallocate(w worker.Worker) {
	p.Lock()
	defer p.Unlock()
	delete(p.workers, w)
}

// List returns a list of all Workers currently
// allocated to the Pool.
func (p *Pool) List() []worker.Worker {
	p.Lock()
	defer p.Unlock()

	var workers []worker.Worker
	for w := range p.workers {
		workers = append(workers, w)
	}
	return workers
}

// Reserve reserves the next available worker to
// start doing work. Once work is complete, the
// worker should be released back to the pool.
func (p *Pool) Reserve() <-chan worker.Worker {
	return p.workerc
}

// Release releases the worker back to the pool
// of available workers.
func (p *Pool) Release(w worker.Worker) bool {
	if !p.IsAllocated(w) {
		return false
	}

	p.workerc <- w
	return true
}
