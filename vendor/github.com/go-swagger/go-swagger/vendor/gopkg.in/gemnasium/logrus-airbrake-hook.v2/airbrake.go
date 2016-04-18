package airbrake // import "gopkg.in/gemnasium/logrus-airbrake-hook.v2"

import (
	"errors"
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
	"gopkg.in/airbrake/gobrake.v2"
)

// Set airbrake.BufSize = <value> _before_ calling NewHook
var BufSize uint = 1024

// AirbrakeHook to send exceptions to an exception-tracking service compatible
// with the Airbrake API.
type airbrakeHook struct {
	Airbrake   *gobrake.Notifier
	noticeChan chan *gobrake.Notice
}

func NewHook(projectID int64, apiKey, env string) *airbrakeHook {
	airbrake := gobrake.NewNotifier(projectID, apiKey)
	airbrake.AddFilter(func(notice *gobrake.Notice) *gobrake.Notice {
		if env == "development" {
			return nil
		}
		notice.Context["environment"] = env
		return notice
	})
	hook := &airbrakeHook{
		Airbrake:   airbrake,
		noticeChan: make(chan *gobrake.Notice, BufSize),
	}
	go hook.fire()
	return hook
}

func (hook *airbrakeHook) Fire(entry *logrus.Entry) error {
	var notifyErr error
	err, ok := entry.Data["error"].(error)
	if ok {
		notifyErr = err
	} else {
		notifyErr = errors.New(entry.Message)
	}
	notice := hook.Airbrake.Notice(notifyErr, nil, 3)
	for k, v := range entry.Data {
		notice.Context[k] = fmt.Sprintf("%s", v)
	}
	// Don't exit before sending the exception
	if entry.Level == logrus.ErrorLevel || entry.Level == logrus.PanicLevel {
		hook.sendNotice(notice)
		return nil
	}
	hook.noticeChan <- notice
	return nil
}

// fire sends errors to airbrake when an entry is available on entryChan
func (hook *airbrakeHook) fire() {
	for {
		notice := <-hook.noticeChan
		hook.sendNotice(notice)
	}
}

func (hook *airbrakeHook) sendNotice(notice *gobrake.Notice) {
	if _, err := hook.Airbrake.SendNotice(notice); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to send error to Airbrake: %v\n", err)
	}
}

func (hook *airbrakeHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.ErrorLevel,
		logrus.FatalLevel,
		logrus.PanicLevel,
	}
}
