package eventbus

import "github.com/drone/drone/common"

type Event struct {
	Build *common.Build `json:"build,omitempty"`
	Repo  *common.Repo  `json:"repo,omitempty"`
	Task  *common.Task  `json:"task,omitempty"`
}

type Bus interface {
	Subscribe(chan *Event)
	Unsubscribe(chan *Event)
	Send(*Event)
}
