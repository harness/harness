package transform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/drone/drone/yaml"
)

func convertTransform(c *yaml.Config, url string) error {
	var buf bytes.Buffer

	// encode yaml in json format
	if err := json.NewEncoder(&buf).Encode(c); err != nil {
		return err
	}
	resp, err := http.Post(url, "application/json", &buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// decode the updated yaml from the body
	if resp.StatusCode == 200 {
		err = json.NewDecoder(resp.Body).Decode(c)
		return err
	}

	// handle error response
	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return fmt.Errorf(string(out))
}

// RemoteTransform makes remote transform requests.
func RemoteTransform(c *yaml.Config, url []string) error {
	if len(url) == 0 {
		return nil
	}

	for _, u := range url {
		if err := convertTransform(c, u); err != nil {
			return err
		}
	}

	return nil
}
