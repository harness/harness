// Package pubsub implements a publish-subscriber messaging system.
package pubsub

import (
	"context"
	"errors"
)

// ErrNotFound is returned when the named topic does not exist.
var ErrNotFound = errors.New("topic not found")

// Message defines a published message.
type Message struct {
	// ID identifies this message.
	ID string `json:"id,omitempty"`

	// Data is the actual data in the entry.
	Data []byte `json:"data"`

	// Labels represents the key-value pairs the entry is lebeled with.
	Labels map[string]string `json:"labels,omitempty"`
}

// Receiver receives published messages.
type Receiver func(Message)

// Publisher defines a mechanism for communicating messages from a group
// of senders, called producers, to a group of consumers.
type Publisher interface {
	// Create creates the named topic.
	Create(c context.Context, topic string) error

	// Publish publishes the message.
	Publish(c context.Context, topic string, message Message) error

	// Subscribe subscribes to the topic. The Receiver function is a callback
	// function that receives published messages.
	Subscribe(c context.Context, topic string, receiver Receiver) error

	// Remove removes the named topic.
	Remove(c context.Context, topic string) error
}

// // global instance of the queue.
// var global = New()
//
// // Set sets the global queue.
// func Set(p Publisher) {
// 	global = p
// }
//
// // Create creates the named topic.
// func Create(c context.Context, topic string) error {
// 	return global.Create(c, topic)
// }
//
// // Publish publishes the message.
// func Publish(c context.Context, topic string, message Message) error {
// 	return global.Publish(c, topic, message)
// }
//
// // Subscribe subscribes to the topic. The Receiver function is a callback
// // function that receives published messages.
// func Subscribe(c context.Context, topic string, receiver Receiver) error {
// 	return global.Subscribe(c, topic, receiver)
// }
//
// // Remove removes the topic.
// func Remove(c context.Context, topic string) error {
// 	return global.Remove(c, topic)
// }
