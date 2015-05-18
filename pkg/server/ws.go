package server

import (
	"io"
	"net/http"
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

	var rc io.ReadCloser

	// if no agent is assigned to the build we
	// should stream the local logs
	if commit.AgentID == 0 {
		rc, err = runner.Logs(build)
		if err != nil {
			c.Fail(404, err)
			return
		}
	} else {
		agent, err := store.Agent(commit.AgentID)
		if err != nil {
			c.Fail(404, err)
			return
		}
		resp, err := http.Get("http://" + agent.Addr)
		if err != nil {
			c.Fail(500, err)
			return
		} else if resp.StatusCode != 200 {
			resp.Body.Close()
			c.AbortWithStatus(resp.StatusCode)
			return
		}
		rc = resp.Body
	}

	defer func() {
		rc.Close()
	}()

	go func() {
		<-c.Writer.CloseNotify()
		rc.Close()
	}()

	rw := &StreamWriter{c.Writer, 0}

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
