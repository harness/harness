package webhook

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/drone/drone/shared/model"
)

type Webhook struct {
	URL     []string `yaml:"urls,omitempty"`
	Success *bool    `yaml:"on_success,omitempty"`
	Failure *bool    `yaml:"on_failure,omitempty"`
}

func (w *Webhook) Send(context *model.Request) error {
	switch {
	case context.Commit.Status == model.StatusSuccess && w.Success != nil && *w.Success == true:
		return w.send(context)
	case context.Commit.Status == model.StatusFailure && w.Failure != nil && *w.Failure == true:
		return w.send(context)
	}

	return nil
}

// helper function to send HTTP requests
func (w *Webhook) send(context *model.Request) error {
	// data will get posted in this format
	data := struct {
		From   string        `json:"from_url"`
		Owner  *model.User   `json:"owner"`
		Repo   *model.Repo   `json:"repository"`
		Commit *model.Commit `json:"commit"`
	}{context.Host, context.User, context.Repo, context.Commit}

	// data json encoded
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

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
