package model

// EventType defines the possible types of build events.
type EventType string

const (
	Enqueued  EventType = "enqueued"
	Started   EventType = "started"
	Finished  EventType = "finished"
	Cancelled EventType = "cancelled"
)

// Event represents a build event.
type Event struct {
	Type  EventType `json:"type"`
	Repo  Repo      `json:"repo"`
	Build Build     `json:"build"`
	Proc  Proc      `json:"proc"`
}
