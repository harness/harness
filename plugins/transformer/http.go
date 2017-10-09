package transformer

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/drone/drone/model"

	"github.com/Sirupsen/logrus"
)

type httpTransformer struct {
	url string
}

func NewHTTP(url string) *httpTransformer {
	return &httpTransformer{url}
}

type httpPayload struct {
	Repo   *model.Repo  `json:"repo"`
	Config []byte `json:"config"`
}

func (t *httpTransformer) Transform(repo *model.Repo, data []byte) ([]byte, error) {
	logrus.Debugf("Doing a transform for %s", repo.FullName)

	jsonPayload, err := json.Marshal(httpPayload{repo, data})
	if err != nil {
		return []byte{}, err
	}

	resp, err := http.Post(t.url, "application/json", bytes.NewReader(jsonPayload))
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	return body, nil
}
