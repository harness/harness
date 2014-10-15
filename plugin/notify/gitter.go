package notify

import (
	"encoding/json"
	"fmt"

	"github.com/drone/drone/shared/model"
)

const (
	gitterEndpoint       = "https://api.gitter.im/v1/rooms/%s/chatMessages"
	gitterStartedMessage = "*Building* %s, commit [%s](%s), author %s"
	gitterSuccessMessage = "*Success* %s, commit [%s](%s), author %s"
	gitterFailureMessage = "*Failed* %s, commit [%s](%s), author %s"
)

type Gitter struct {
	RoomID  string `yaml:"room_id,omitempty"`
	Token   string `yaml:"token,omitempty"`
	Started bool   `yaml:"on_started,omitempty"`
	Success bool   `yaml:"on_success,omitempty"`
	Failure bool   `yaml:"on_failure,omitempty"`
}

func (g *Gitter) Send(context *model.Request) error {
	switch {
	case context.Commit.Status == model.StatusStarted && g.Started:
		return g.sendStarted(context)
	case context.Commit.Status == model.StatusSuccess && g.Success:
		return g.sendSuccess(context)
	case context.Commit.Status == model.StatusFailure && g.Failure:
		return g.sendFailure(context)
	}

	return nil
}

func (g *Gitter) getMessage(context *model.Request, message string) string {
	url := getBuildUrl(context)
	return fmt.Sprintf(message, context.Repo.Name, context.Commit.ShaShort(), url, context.Commit.Author)
}

func (g *Gitter) sendStarted(context *model.Request) error {
	return g.send(g.getMessage(context, gitterStartedMessage))
}

func (g *Gitter) sendSuccess(context *model.Request) error {
	return g.send(g.getMessage(context, gitterSuccessMessage))
}

func (g *Gitter) sendFailure(context *model.Request) error {
	return g.send(g.getMessage(context, gitterFailureMessage))
}

// helper function to send HTTP requests
func (g *Gitter) send(msg string) error {
	// data will get posted in this format
	data := struct {
		Text string `json:"text"`
	}{msg}

	// data json encoded
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// send payload
	url := fmt.Sprintf(gitterEndpoint, g.RoomID)

	// create headers
	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	headers["Authorization"] = fmt.Sprintf("Bearer %s", g.Token)

	return sendJson(url, payload, headers)
}
