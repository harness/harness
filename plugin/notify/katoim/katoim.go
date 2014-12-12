package katoim

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/drone/drone/shared/model"
)

const (
	katoimEndpoint       = "https://api.kato.im/rooms/%s/simple"
	katoimStartedMessage = "*Building* %s, commit [%s](%s), author %s"
	katoimSuccessMessage = "*Success* %s, commit [%s](%s), author %s"
	katoimFailureMessage = "*Failed* %s, commit [%s](%s), author %s"

	NotifyTrue   = "true"
	NotifyFalse  = "false"
	NotifyOn     = "on"
	NotifyOff    = "off"
	NotifyNever  = "never"
	NotifyAlways = "always"
)

type KatoIM struct {
	RoomID  string `yaml:"room_id,omitempty"`
	Started string `yaml:"on_started,omitempty"`
	Success string `yaml:"on_success,omitempty"`
	Failure string `yaml:"on_failure,omitempty"`
}

func (k *KatoIM) Send(context *model.Request) error {
	switch {
	case context.Commit.Status == model.StatusStarted:
		return k.sendStarted(context)
	case context.Commit.Status == model.StatusSuccess:
		return k.sendSuccess(context)
	case context.Commit.Status == model.StatusFailure:
		return k.sendFailure(context)
	}

	return nil
}

func (k *KatoIM) getMessage(context *model.Request, message string) string {
	url := getBuildUrl(context)
	return fmt.Sprintf(message, context.Repo.Name, context.Commit.ShaShort(), url, context.Commit.Author)
}

// sendStarted disabled by default
func (k *KatoIM) sendStarted(context *model.Request) error {
	switch k.Started {
	case NotifyTrue, NotifyAlways, NotifyOn:
		return k.send(k.getMessage(context, katoimStartedMessage), "yellow")
	default:
		return nil
	}
}

// sendSuccess enabled by default
func (k *KatoIM) sendSuccess(context *model.Request) error {
	switch k.Success {
	case NotifyFalse, NotifyNever, NotifyOff:
		return nil
	case NotifyTrue, NotifyAlways, NotifyOn, "":
		return k.send(k.getMessage(context, katoimSuccessMessage), "green")
	default:
		return nil
	}
}

// sendFailure enabled by default
func (k *KatoIM) sendFailure(context *model.Request) error {
	switch k.Failure {
	case NotifyFalse, NotifyNever, NotifyOff:
		return nil
	case NotifyTrue, NotifyAlways, NotifyOn, "":
		return k.send(k.getMessage(context, katoimFailureMessage), "red")
	default:
		return nil
	}
}

// helper function to send HTTP requests
func (k *KatoIM) send(msg, color string) error {
	// data will get posted in this format
	data := struct {
		Text     string `json:"text"`
		Color    string `json:"color"`
		Renderer string `json:"renderer"`
		From     string `json:"from"`
	}{msg, color, "markdown", "Drone"}

	// data json encoded
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// send payload
	url := fmt.Sprintf(katoimEndpoint, k.RoomID)

	// create headers
	headers := make(map[string]string)
	headers["Accept"] = "application/json"

	return sendJson(url, payload, headers)
}

func getBuildUrl(context *model.Request) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s/%s", context.Host, context.Repo.Host, context.Repo.Owner, context.Repo.Name, context.Commit.Branch, context.Commit.Sha)
}

// helper fuction to sent HTTP Post requests
// with JSON data as the payload.
func sendJson(url string, payload []byte, headers map[string]string) error {
	client := &http.Client{}
	buf := bytes.NewBuffer(payload)

	req, err := http.NewRequest("POST", url, buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if headers != nil {
		for k, v := range headers {
			req.Header.Add(k, v)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
