package queue

//go:generate mockery -name Queue -output mock -case=underscore

import (
	"errors"

	"golang.org/x/net/context"
)

// ErrNotFound indicates the requested work item does not
// exist in the queue.
var ErrNotFound = errors.New("queue item not found")

type Queue interface {
	// Publish inserts work at the tail of this queue, waiting for
	// space to become available if the queue is full.
	Publish(*Work) error

	// Remove removes the specified work item from this queue,
	// if it is present.
	Remove(*Work) error

	// PullClose retrieves and removes the head of this queue,
	// waiting if necessary until work becomes available.
	Pull() *Work

	// PullClose retrieves and removes the head of this queue,
	// waiting if necessary until work becomes available. The
	// CloseNotifier should be provided to clone the channel
	// if the subscribing client terminates its connection.
	PullClose(CloseNotifier) *Work
}

// Publish inserts work at the tail of this queue, waiting for
// space to become available if the queue is full.
func Publish(c context.Context, w *Work) error {
	return FromContext(c).Publish(w)
}

// Remove removes the specified work item from this queue,
// if it is present.
func Remove(c context.Context, w *Work) error {
	return FromContext(c).Remove(w)
}

// Pull retrieves and removes the head of this queue,
// waiting if necessary until work becomes available.
func Pull(c context.Context) *Work {
	return FromContext(c).Pull()
}

// PullClose retrieves and removes the head of this queue,
// waiting if necessary until work becomes available. The
// CloseNotifier should be provided to clone the channel
// if the subscribing client terminates its connection.
func PullClose(c context.Context, cn CloseNotifier) *Work {
	return FromContext(c).PullClose(cn)
}

// CloseNotifier defines a datastructure that is capable of notifying
// a subscriber when its connection is closed.
type CloseNotifier interface {
	// CloseNotify returns a channel that receives a single value
	// when the client connection has gone away.
	CloseNotify() <-chan bool
}
