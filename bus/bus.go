package bus

//go:generate mockery -name Bus -output mock -case=underscore

import "golang.org/x/net/context"

// Bus represents an event bus implementation that
// allows a publisher to broadcast Event notifications
// to a list of subscribers.
type Bus interface {
	// Publish broadcasts an event to all subscribers.
	Publish(*Event)

	// Subscribe adds the channel to the list of
	// subscribers. Each subscriber in the list will
	// receive broadcast events.
	Subscribe(chan *Event)

	// Unsubscribe removes the channel from the list
	// of subscribers.
	Unsubscribe(chan *Event)
}

// Publish broadcasts an event to all subscribers.
func Publish(c context.Context, event *Event) {
	FromContext(c).Publish(event)
}

// Subscribe adds the channel to the list of
// subscribers. Each subscriber in the list will
// receive broadcast events.
func Subscribe(c context.Context, eventc chan *Event) {
	FromContext(c).Subscribe(eventc)
}

// Unsubscribe removes the channel from the
// list of subscribers.
func Unsubscribe(c context.Context, eventc chan *Event) {
	FromContext(c).Unsubscribe(eventc)
}
