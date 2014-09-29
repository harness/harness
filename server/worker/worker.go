package worker

import (
	"code.google.com/p/go.net/context"
)

type Worker interface {
	Do(context.Context, *Work)
}

// Do retrieves a worker from the session and uses
// it to get work done.
func Do(c context.Context, w *Work) {
	FromContext(c).Do(c, w)
}
