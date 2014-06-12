package notify

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/drone/drone/server/resource/commit"
	"github.com/drone/drone/server/resource/repo"
	"github.com/drone/drone/server/resource/user"
)

type Webhook struct {
	URL     []string `yaml:"urls,omitempty"`
	Success bool     `yaml:"on_success,omitempty"`
	Failure bool     `yaml:"on_failure,omitempty"`
}

func (w *Webhook) Send(context *Context) error {
	switch {
	case context.Commit.Status == "Success" && w.Success:
		return w.send(context)
	case context.Commit.Status == "Failure" && w.Failure:
		return w.send(context)
	}

	return nil
}

// helper function to send HTTP requests
func (w *Webhook) send(context *Context) error {
	// data will get posted in this format
	data := struct {
		Owner  *user.User     `json:"owner"`
		Repo   *repo.Repo     `json:"repository"`
		Commit *commit.Commit `json:"commit"`
	}{context.User, context.Repo, context.Commit}

	// data json encoded
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// loop through and email recipients
	for _, url := range w.URL {
		go sendJson(url, payload)
	}
	return nil
}

// helper fuction to sent HTTP Post requests
// with JSON data as the payload.
func sendJson(url string, payload []byte) {
	buf := bytes.NewBuffer(payload)
	resp, err := http.Post(url, "application/json", buf)
	if err != nil {
		return
	}
	resp.Body.Close()
}
