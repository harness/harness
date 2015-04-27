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

	// PullClose retrieves and removes the head of this queue,
	// waiting if necessary until work becomes available. The
	// CloseNotifier should be provided to clone the channel
	// if the subscribing client terminates its connection.
	PullClose(CloseNotifier) *Work

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

type CloseNotifier interface {
	// CloseNotify returns a channel that receives a single value
	// when the client connection has gone away.
	CloseNotify() <-chan bool
}
