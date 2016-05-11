package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
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

func NewClientLogger(client client.Client, id int64, rc io.ReadCloser, wc io.WriteCloser) LoggerFunc {
	var once sync.Once
	return func(line *build.Line) {
		// annoying hack to only start streaming once the first line is written
		once.Do(func() {
			go func() {
				err := client.Stream(id, rc)
				if err != nil && err != io.ErrClosedPipe {
					logrus.Errorf("Error streaming build logs. %s", err)
				}
			}()
		})

		linejson, _ := json.Marshal(line)
		wc.Write(linejson)
		wc.Write([]byte{'\n'})
	}
}
