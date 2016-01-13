package client

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	BaseUrl string
	ApiPath string
	Token   string
	Client  *http.Client
}

func New(baseUrl, apiPath, token string, skipVerify bool) *Client {
	config := &tls.Config{InsecureSkipVerify: skipVerify}
	tr := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: config,
	}
	client := &http.Client{Transport: tr}

	return &Client{
		BaseUrl: baseUrl,
		ApiPath: apiPath,
		Token:   token,
		Client:  client,
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

	u = c.BaseUrl + c.ApiPath + u + "?" + query_params.Encode()
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
		return nil, fmt.Errorf("Error while building gitlab request")
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))

	if len(opaque) > 0 {
		req.URL.Opaque = opaque
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Client.Do error: %q", err)
	}
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("%s", err)
	}

	if resp.StatusCode >= 400 {
		err = fmt.Errorf("*Gitlab.buildAndExecRequest failed: <%d> %s", resp.StatusCode, req.URL)
	}

	return contents, err
}
