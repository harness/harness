package queue

import (
	"io"

	common "github.com/drone/drone/pkg/types"
)

// Work represents an item for work to be
// processed by a worker.
type Work struct {
	User    *common.User    `json:"user"`
	Repo    *common.Repo    `json:"repo"`
	Job 	*common.Job		`json:"job"`
	Keys    *common.Keypair `json:"keypair"`
	Netrc   *common.Netrc   `json:"netrc"`
	Yaml    []byte          `json:"yaml"`
	Env     []string        `json:"environment"`
	Plugins []string        `json:"plugins"`
}

// represents a worker that has connected
// to the system in order to perform work
type Worker struct {
	Name      string
	Addr      string
	IsHealthy bool
}

// Ping pings to worker to verify it is
// available and in good health.
func (w *Worker) Ping() (bool, error) {
	return false, nil
}

// Logs fetches the logs for a work item.
func (w *Worker) Logs() (io.Reader, error) {
	return nil, nil
}

// Cancel cancels a work item.
func (w *Worker) Cancel() error {
	return nil
}

// type Monitor struct {
// 	manager *Manager
// }

// func NewMonitor(manager *Manager) *Monitor {
// 	return &Monitor{manager}
// }

// // start is a helper function that is used to monitor
// // all registered workers and ensure they are in a
// // healthy state.
// func (m *Monitor) Start() {
// 	ticker := time.NewTicker(1 * time.Hour)
// 	go func() {
// 		for {
// 			select {
// 			case <-ticker.C:
// 				workers := m.manager.Workers()
// 				for _, worker := range workers {
// 					// ping the worker to make sure it is
// 					// available and still accepting builds.
// 					if _, err := worker.Ping(); err != nil {
// 						m.manager.SetHealth(worker, false)
// 					} else {
// 						m.manager.SetHealth(worker, true)
// 					}
// 				}
// 			}
// 		}
// 	}
