package queue

//go:generate mockery -name Queue -output mock -case=underscore

import (
	"errors"

	"golang.org/x/net/context"
)

// ErrNotFound indicates the requested work item does not
// exist in the queue.
var ErrNotFound = errors.New("queue item not found")

// Queue represents a worker queue.
type Queue interface {
	// Publish inserts work at the tail of this queue, waiting for
	// space to become available if the queue is full.
	Publish(*Work) error

	// Remove removes the specified job ID from this queue,
	// if it is present.
	Remove(int64) error

	// PullClose retrieves and removes the head of this queue,
	// waiting if necessary until work becomes available.
	Pull() *Work

	// PullClose retrieves and removes the head of this queue,
	// waiting if necessary until work becomes available. The
	// CloseNotifier should be provided to clone the channel
	// if the subscribing client terminates its connection.
	PullClose(CloseNotifier) *Work

	// IndexOf retrieves the current position within the queue of
	// the job ID or -1 if the work is not found.
	IndexOf(int64) int

	// Length retrieves the length of the queue.
	Length() int
}

// Publish inserts work at the tail of this queue, waiting for
// space to become available if the queue is full.
func Publish(c context.Context, w *Work) error {
	return FromContext(c).Publish(w)
}

// Remove removes the specified job ID from this queue,
// if it is present.
func Remove(c context.Context, id int64) error {
	return FromContext(c).Remove(id)
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

// IndexOf retrieves the current position within the queue of
// the job ID or -1 if the work is not found.
func IndexOf(c context.Context, id int64) int {
	return FromContext(c).IndexOf(id)
}

// Length retrieves the length of the queue.
func Length(c context.Context) int {
	return FromContext(c).Length()
}

// CloseNotifier defines a datastructure that is capable of notifying
// a subscriber when its connection is closed.
type CloseNotifier interface {
	// CloseNotify returns a channel that receives a single value
	// when the client connection has gone away.
	CloseNotify() <-chan bool
}
