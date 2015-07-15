package citadel

import "time"

type Event struct {
	Type      string     `json:"type,omitempty"`
	Container *Container `json:"container,omitempty"`
	Engine    *Engine    `json:"engine,omitempty"`
	Time      time.Time  `json:"time,omitempty"`
}

type EventHandler interface {
	Handle(*Event) error
}
