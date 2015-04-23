package queue

type Queue interface {
	// Publish inserts work at the tail of this queue, waiting for
	// space to become available if the queue is full.
	Publish(*Work) error

	// Remove removes the specified work item from this queue,
	// if it is present.
	Remove(*Work) error

	// Pull retrieves and removes the head of this queue, waiting
	// if necessary until work becomes available.
	Pull() *Work

	// PullAck retrieves and removes the head of this queue, waiting
	// if necessary until work becomes available. Items pull from the
	// queue that aren't acknowledged will be pushed back to the queue
	// again when the default acknowledgement deadline is reached.
	PullAck() *Work

	// Ack acknowledges an item in the queue was processed.
	Ack(*Work) error

	// Items returns a slice containing all of the work in this
	// queue, in proper sequence.
	Items() []*Work
}

// type Manager interface {
// 	// Register registers a worker that has signed
// 	// up to accept work.
// 	Register(*Worker)

// 	// Unregister unregisters a worker that should no
// 	// longer be accepting work.
// 	Unregister(*Worker)

// 	// Assign assigns work to a worker.
// 	Assign(*Work, *Worker)

// 	// Unassign unassigns work from a worker.
// 	Unassign(*Work, *Worker)

// 	// Work returns a list of all work that is
// 	// currently in progress.
// 	Work() []*Work

// 	// Worker retrieves a worker by name.
// 	Worker(string) *Worker

// 	// Workers returns a slice containing all workers
// 	// registered with the manager.
// 	Workers() []*Worker
// }
