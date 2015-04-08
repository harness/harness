package eventbus

import "github.com/bradrydzewski/drone/common"

type Event struct {
	Build *common.Build
	Repo  *common.Repo
	Task  *common.Task
}

type Bus interface {
	Subscribe(chan *Event)
	Unsubscribe(chan *Event)
	Send(*Event)
}
