package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/drone/drone/build"
	"github.com/drone/drone/client"
	"github.com/drone/drone/queue"
)

// UpdateFunc handles buid pipeline status updates.
type UpdateFunc func(*queue.Work)

// LoggerFunc handles buid pipeline logging updates.
type LoggerFunc func(*build.Line)

var NoopUpdateFunc = func(*queue.Work) {}

var TermLoggerFunc = func(line *build.Line) {
	fmt.Println(line)
}

// NewClientUpdater returns an updater that sends updated build details
// to the drone server.
func NewClientUpdater(client client.Client) UpdateFunc {
	return func(w *queue.Work) {
		for {
			err := client.Push(w)
			if err == nil {
				return
			}
			logrus.Errorf("Error updating %s/%s#%d.%d. Retry in 30s. %s",
				w.Repo.Owner, w.Repo.Name, w.Build.Number, w.Job.Number, err)
			logrus.Infof("Retry update in 30s")
			time.Sleep(time.Second * 30)
		}
	}
}

func NewClientLogger(w io.Writer) LoggerFunc {
	return func(line *build.Line) {
		linejson, _ := json.Marshal(line)
		w.Write(linejson)
		w.Write([]byte{'\n'})
	}
}
