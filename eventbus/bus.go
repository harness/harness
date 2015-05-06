package eventbus

const (
	EventRepo  = "repo"
	EventUser  = "user"
	EventAgent = "agent"
)

type Event struct {
	Kind string
	Name string
	Msg  []byte
}

type Bus interface {
	Subscribe(chan *Event)
	Unsubscribe(chan *Event)
	Send(*Event)
}
