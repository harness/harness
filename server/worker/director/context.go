package director

import (
	"code.google.com/p/go.net/context"
	"github.com/drone/drone/server/worker"
)

// NewContext returns a Context whose Value method returns
// the director.
func NewContext(parent context.Context, w worker.Worker) context.Context {
	return worker.NewContext(parent, w)
}
