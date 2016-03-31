package web

import (
	"io"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/docker/docker/pkg/stdcopy"
	"github.com/drone/drone/engine"
	"github.com/drone/drone/router/middleware/context"
	"github.com/drone/drone/router/middleware/session"
	"github.com/drone/drone/store"

	log "github.com/Sirupsen/logrus"

	"github.com/manucorporat/sse"
)

// GetRepoEvents will upgrade the connection to a Websocket and will stream
// event updates to the browser.
func GetRepoEvents(c *gin.Context) {
	engine_ := context.Engine(c)
	repo := session.Repo(c)
	c.Writer.Header().Set("Content-Type", "text/event-stream")

	eventc := make(chan *engine.Event, 1)
	engine_.Subscribe(eventc)
	defer func() {
		engine_.Unsubscribe(eventc)
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
			if event.Name == repo.FullName {
				log.Debugf("received message %s", event.Name)
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

	engine_ := context.Engine(c)
	repo := session.Repo(c)
	buildn, _ := strconv.Atoi(c.Param("build"))
	jobn, _ := strconv.Atoi(c.Param("number"))

	c.Writer.Header().Set("Content-Type", "text/event-stream")

	build, err := store.GetBuildNumber(c, repo, buildn)
	if err != nil {
		log.Debugln("stream cannot get build number.", err)
		c.AbortWithError(404, err)
		return
	}
	job, err := store.GetJobNumber(c, build, jobn)
	if err != nil {
		log.Debugln("stream cannot get job number.", err)
		c.AbortWithError(404, err)
		return
	}
	node, err := store.GetNode(c, job.NodeID)
	if err != nil {
		log.Debugln("stream cannot get node.", err)
		c.AbortWithError(404, err)
		return
	}

	rc, err := engine_.Stream(build.ID, job.ID, node)
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	defer func() {
		rc.Close()
	}()

	go func() {
		defer func() {
			recover()
		}()
		<-c.Writer.CloseNotify()
		rc.Close()
	}()

	rw := &StreamWriter{c.Writer, 0}

	stdcopy.StdCopy(rw, rw, rc)
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
