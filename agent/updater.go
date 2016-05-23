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

func NewStreamLogger(stream client.StreamWriter, w io.Writer, limit int64) LoggerFunc {
	var err error
	var size int64
	return func(line *build.Line) {

		if size > limit {
			return
		}

		// TODO remove this double-serialization
		linejson, _ := json.Marshal(line)
		w.Write(linejson)
		w.Write([]byte{'\n'})

		if err = stream.WriteJSON(line); err != nil {
			logrus.Errorf("Error streaming build logs. %s", err)
		}

		size += int64(len(line.Out))
	}
}

func NewClientLogger(client client.Client, id int64, rc io.ReadCloser, wc io.WriteCloser, limit int64) LoggerFunc {
	var once sync.Once
	var size int64
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

		if size > limit {
			return
		}

		linejson, _ := json.Marshal(line)
		wc.Write(linejson)
		wc.Write([]byte{'\n'})

		size += int64(len(line.Out))
	}
}
