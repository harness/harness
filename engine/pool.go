package engine

import (
	"sync"

	"github.com/drone/drone/model"
)

type pool struct {
	sync.Mutex
	nodes map[*model.Node]bool
	nodec chan *model.Node
}

func newPool() *pool {
	return &pool{
		nodes: make(map[*model.Node]bool),
		nodec: make(chan *model.Node, 999),
	}
}

// Allocate allocates a node to the pool to
// be available to accept work.
func (p *pool) allocate(n *model.Node) bool {
	if p.isAllocated(n) {
		return false
	}

	p.Lock()
	p.nodes[n] = true
	p.Unlock()

	p.nodec <- n
	return true
}

// IsAllocated is a helper function that returns
// true if the node is currently allocated to
// the pool.
func (p *pool) isAllocated(n *model.Node) bool {
	p.Lock()
	defer p.Unlock()
	_, ok := p.nodes[n]
	return ok
}

// Deallocate removes the node from the pool of
// available nodes. If the node is currently
// reserved and performing work it will finish,
// but no longer be given new work.
func (p *pool) deallocate(n *model.Node) {
	p.Lock()
	defer p.Unlock()
	delete(p.nodes, n)
}

// List returns a list of all model.Nodes currently
// allocated to the pool.
func (p *pool) list() []*model.Node {
	p.Lock()
	defer p.Unlock()

	var nodes []*model.Node
	for n := range p.nodes {
		nodes = append(nodes, n)
	}
	return nodes
}

// Reserve reserves the next available node to
// start doing work. Once work is complete, the
// node should be released back to the pool.
func (p *pool) reserve() <-chan *model.Node {
	return p.nodec
}

// Release releases the node back to the pool
// of available nodes.
func (p *pool) release(n *model.Node) bool {
	if !p.isAllocated(n) {
		return false
	}

	p.nodec <- n
	return true
}
