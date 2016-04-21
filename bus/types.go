package bus

import "github.com/drone/drone/model"

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
	Type  EventType   `json:"type"`
	Repo  model.Repo  `json:"repo"`
	Build model.Build `json:"build"`
	Job   model.Job   `json:"job"`
}

// NewEvent creates a new Event for the build, using copies of
// the build data to avoid possible mutation or race conditions.
func NewEvent(t EventType, r *model.Repo, b *model.Build, j *model.Job) *Event {
	return &Event{
		Type:  t,
		Repo:  *r,
		Build: *b,
		Job:   *j,
	}
}

func NewBuildEvent(t EventType, r *model.Repo, b *model.Build) *Event {
	return &Event{
		Type:  t,
		Repo:  *r,
		Build: *b,
	}
}
