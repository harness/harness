// Package github implements a simple client to consume gitlab API.
package gogitlab

import (
	"bytes"
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

const (
	dasboard_feed_path = "/dashboard.atom"
)

type Gitlab struct {
	BaseUrl      string
	ApiPath      string
	RepoFeedPath string
	Token        string
	Bearer       bool
	Client       *http.Client
}

const (
	dateLayout = "2006-01-02T15:04:05-07:00"
)

var (
	skipCertVerify = flag.Bool("gitlab.skip-cert-check", false,
		`If set to true, gitlab client will skip certificate checking for https, possibly exposing your system to MITM attack.`)
)

func NewGitlab(baseUrl, apiPath, token string) *Gitlab {
	return NewGitlabCert(baseUrl, apiPath, token, *skipCertVerify)
}

func NewGitlabCert(baseUrl, apiPath, token string, skipVerify bool) *Gitlab {
	config := &tls.Config{InsecureSkipVerify: skipVerify}
	tr := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: config,
	}
	client := &http.Client{Transport: tr}

	return &Gitlab{
		BaseUrl: baseUrl,
		ApiPath: apiPath,
		Token:   token,
		Client:  client,
	}
}

func (g *Gitlab) ResourceUrl(url string, params map[string]string) string {

	if params != nil {
		for key, val := range params {
			url = strings.Replace(url, key, encodeParameter(val), -1)
		}
	}

	url = g.BaseUrl + g.ApiPath + url
	if !g.Bearer {
		url = url + "?private_token=" + g.Token
	}
	return url
}

func (g *Gitlab) buildAndExecRequest(method, url string, body []byte) ([]byte, error) {

	var req *http.Request
	var err error

	if body != nil {
		reader := bytes.NewReader(body)
		req, err = http.NewRequest(method, url, reader)
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		panic("Error while building gitlab request")
	}

	if g.Bearer {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.Token))
	}

	resp, err := g.Client.Do(req)
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

func (g *Gitlab) ResourceUrlQuery(u string, params, query map[string]string) string {
	if params != nil {
		for key, val := range params {
			u = strings.Replace(u, key, encodeParameter(val), -1)
		}
	}

	query_params := url.Values{}
	if !g.Bearer {
		query_params.Add("private_token", g.Token)
	}

	if query != nil {
		for key, val := range query {
			query_params.Set(key, val)
		}
	}

	u = g.BaseUrl + g.ApiPath + u + "?" + query_params.Encode()
	return u

}

func (g *Gitlab) ResourceUrlRaw(u string, params map[string]string) (string, string) {

	if params != nil {
		for key, val := range params {
			u = strings.Replace(u, key, encodeParameter(val), -1)
		}
	}

	path := u
	u = g.BaseUrl + g.ApiPath + path
	if !g.Bearer {
		u = u + "?private_token=" + g.Token
	}

	p, err := url.Parse(u)
	if err != nil {
		return u, ""
	}
	opaque := "//" + p.Host + p.Path
	return u, opaque
}

func (g *Gitlab) buildAndExecRequestRaw(method, url, opaque string, body []byte) ([]byte, error) {

	var req *http.Request
	var err error

	if body != nil {
		reader := bytes.NewReader(body)
		req, err = http.NewRequest(method, url, reader)
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		panic("Error while building gitlab request")
	}

	if g.Bearer {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.Token))
	}

	if len(opaque) > 0 {
		req.URL.Opaque = opaque
	}

	resp, err := g.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Client.Do error: %q", err)
	}
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("%s", err)
	}

	if resp.StatusCode >= 400 {
		err = fmt.Errorf("*Gitlab.buildAndExecRequestRaw failed: <%d> %s", resp.StatusCode, req.URL)
	}

	return contents, err
}
