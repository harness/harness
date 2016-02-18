package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	BaseUrl string
	Token   string
	Client  *http.Client
}

func New(baseUrl, token string, skipVerify bool) *Client {
	return &Client{
		BaseUrl: baseUrl,
		Token:   token,
		Client: &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: skipVerify,
				},
			},
		},
	}
}

func (c *Client) ResourceUrl(u string, params, query QMap) (string, string) {
	if params != nil {
		for key, val := range params {
			u = strings.Replace(u, key, encodeParameter(val), -1)
		}
	}

	query_params := url.Values{}

	if query != nil {
		for key, val := range query {
			query_params.Set(key, val)
		}
	}

	query_params.Set("access_token", c.Token)

	u = c.BaseUrl + u + "?" + query_params.Encode()
	p, err := url.Parse(u)

	if err != nil {
		return u, ""
	}

	opaque := "//" + p.Host + p.Path
	return u, opaque
}

func (c *Client) Do(method, url, opaque string, body []byte) ([]byte, error) {
	var req *http.Request
	var err error

	if body != nil {
		reader := bytes.NewReader(body)
		req, err = http.NewRequest(method, url, reader)
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("Error while building request")
	}

	if len(opaque) > 0 {
		req.URL.Opaque = opaque
	}

	resp, err := c.Client.Do(req)

	if err != nil {
		return nil, fmt.Errorf("Failed to request. %s", err)
	}

	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		fmt.Printf("Failed to parse response. %s", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Got bad response code <%d> on %s", resp.StatusCode, req.URL)
	}

	var result struct {
		Code    string          `json:"error_code"`
		Info    string          `json:"error_info"`
		Content json.RawMessage `json:"result"`
	}

	if err := json.Unmarshal(contents, &result); err != nil {
		return nil, fmt.Errorf("Failed to parse JSON. %s", err)
	}

	if result.Code != "" {
		return nil, fmt.Errorf("Got a bad response code. %s", result.Info)
	}

	return result.Content, err
}
