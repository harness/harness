package eventbus

import "github.com/drone/drone/Godeps/_workspace/src/github.com/citadel/citadel"

type EventBus struct {
	engines  map[string]*citadel.Engine
	handlers map[string][]citadel.EventHandler
}

func New(engines ...*citadel.Engine) (*EventBus, error) {
	bus := &EventBus{
		engines:  make(map[string]*citadel.Engine),
		handlers: make(map[string][]citadel.EventHandler),
	}

	for _, e := range engines {
		bus.engines[e.ID] = e
	}

	return bus, nil
}

func (b *EventBus) AddHandler(eventType string, h citadel.EventHandler) error {
	b.handlers[eventType] = append(b.handlers[eventType], h)

	return nil
}

func (b *EventBus) Handle(event *citadel.Event) error {
	for tpe, l := range b.handlers {
		if tpe == "*" || tpe == event.Type {
			for _, h := range l {
				if err := h.Handle(event); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
