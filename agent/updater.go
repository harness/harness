package agent

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/drone/drone/build"
	"github.com/drone/drone/model"
	"github.com/drone/mq/logger"
	"github.com/drone/mq/stomp"
)

// UpdateFunc handles buid pipeline status updates.
type UpdateFunc func(*model.Work)

// LoggerFunc handles buid pipeline logging updates.
type LoggerFunc func(*build.Line)

var NoopUpdateFunc = func(*model.Work) {}

var TermLoggerFunc = func(line *build.Line) {
	fmt.Println(line)
}

// NewClientUpdater returns an updater that sends updated build details
// to the drone server.
func NewClientUpdater(client *stomp.Client) UpdateFunc {
	return func(w *model.Work) {
		err := client.SendJSON("/queue/updates", w)
		if err != nil {
			logger.Warningf("Error updating %s/%s#%d.%d. %s",
				w.Repo.Owner, w.Repo.Name, w.Build.Number, w.Job.Number, err)
		}
		if w.Job.Status != model.StatusRunning {
			var dest = fmt.Sprintf("/topic/logs.%d", w.Job.ID)
			var opts = []stomp.MessageOption{
				stomp.WithHeader("eof", "true"),
				stomp.WithRetain("all"),
			}

			if err := client.Send(dest, []byte("eof"), opts...); err != nil {
				logger.Warningf("Error sending eof %s/%s#%d.%d. %s",
					w.Repo.Owner, w.Repo.Name, w.Build.Number, w.Job.Number, err)
			}
		}
	}
}

func NewClientLogger(client *stomp.Client, id int64, limit int64) LoggerFunc {

	var size int64
	var dest = fmt.Sprintf("/topic/logs.%d", id)
	var opts = []stomp.MessageOption{
		stomp.WithRetain("all"),
	}

	return func(line *build.Line) {
		if size > limit {
			return
		}
		if err := client.SendJSON(dest, line, opts...); err != nil {
			logrus.Errorf("Error streaming build logs. %s", err)
		}

		size += int64(len(line.Out))
	}
}
