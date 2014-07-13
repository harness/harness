package worker

import (
	"github.com/drone/drone/shared/model"
)

// http://nesv.github.io/golang/2014/02/25/worker-queues-in-go.html

type Dispatch struct {
	requests chan *model.Request
	workers  chan chan *model.Request
	quit     chan bool
}

func NewDispatch(requests chan *model.Request, workers chan chan *model.Request) *Dispatch {
	return &Dispatch{
		requests: requests,
		workers:  workers,
		quit:     make(chan bool),
	}
}

// Start tells the dispatcher to start listening
// for work requests and dispatching to workers.
func (d *Dispatch) Start() {
	go func() {
		for {
			select {
			// pickup a request from the queue
			case request := <-d.requests:
				go func() {
					// find an available worker and
					// send the request to that worker
					worker := <-d.workers
					worker <- request
				}()
			// listen for a signal to exit
			case <-d.quit:
				return
			}
		}
	}()

}

// Stop tells the dispatcher to stop listening for new
// work requests.
func (d *Dispatch) Stop() {
	go func() { d.quit <- true }()
}
