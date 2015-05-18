package server

import (
	"io"
	"strconv"

	"github.com/drone/drone/pkg/bus"

	log "github.com/Sirupsen/logrus"
	"github.com/drone/drone/pkg/docker"
	"github.com/gin-gonic/gin"
	"github.com/manucorporat/sse"
)

// GetRepoEvents will upgrade the connection to a Websocket and will stream
// event updates to the browser.
func GetRepoEvents(c *gin.Context) {
	bus_ := ToBus(c)
	repo := ToRepo(c)
	c.Writer.Header().Set("Content-Type", "text/event-stream")

	eventc := make(chan *bus.Event, 1)
	bus_.Subscribe(eventc)
	defer func() {
		bus_.Unsubscribe(eventc)
		close(eventc)
		log.Infof("closed event stream")
	}()

	c.Stream(func(w io.Writer) bool {
		select {
		case event := <-eventc:
			if event == nil {
				log.Infof("nil event received")
				return false
			}
			if event.Kind == bus.EventRepo &&
				event.Name == repo.FullName {
				sse.Encode(w, sse.Event{
					Event: "message",
					Data:  string(event.Msg),
				})
			}
		case <-c.Writer.CloseNotify():
			return false
		}
		return true
	})
}

func GetStream(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	runner := ToRunner(c)
	commitseq, _ := strconv.Atoi(c.Params.ByName("build"))
	buildseq, _ := strconv.Atoi(c.Params.ByName("number"))

	c.Writer.Header().Set("Content-Type", "text/event-stream")

	commit, err := store.CommitSeq(repo, commitseq)
	if err != nil {
		c.Fail(404, err)
		return
	}
	build, err := store.BuildSeq(commit, buildseq)
	if err != nil {
		c.Fail(404, err)
		return
	}

	rc, err := runner.Logs(build)
	if err != nil {
		c.Fail(404, err)
		return
	}
	go func() {
		<-c.Writer.CloseNotify()
		rc.Close()
	}()

	rw := &StreamWriter{c.Writer, 0}

	defer func() {
		log.Infof("closed log stream")
		rc.Close()
	}()

	docker.StdCopy(rw, rw, rc)
}

type StreamWriter struct {
	writer gin.ResponseWriter
	count  int
}

func (w *StreamWriter) Write(data []byte) (int, error) {
	var err = sse.Encode(w.writer, sse.Event{
		Id:    strconv.Itoa(w.count),
		Event: "message",
		Data:  string(data),
	})
	w.writer.Flush()
	w.count += len(data)
	return len(data), err
}
