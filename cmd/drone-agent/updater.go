package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	//logs "github.com/Sirupsen/logrus"
	common "github.com/drone/drone/pkg/types"
)

type updater struct {
	addr  string
	token string
}

func (u *updater) SetCommit(user *common.User, r *common.Repo, c *common.Commit) error {
	url_, err := url.Parse(addr)
	if err != nil {
		return err
	}
	url_.Path = fmt.Sprintf("/api/queue/push/%s/%v", r.FullName, c.Sequence)
	var body bytes.Buffer
	json.NewEncoder(&body).Encode(c)
	resp, err := http.Post(url_.String(), "application/json", &body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("Error pushing task state. Code %d", resp.StatusCode)
	}
	return nil
}

func (u *updater) SetBuild(r *common.Repo, c *common.Commit, b *common.Build) error {
	url_, err := url.Parse(u.addr)
	if err != nil {
		return err
	}

	url_.Path = fmt.Sprintf("/api/queue/push/%s", r.FullName)
	var body bytes.Buffer
	json.NewEncoder(&body).Encode(b)
	resp, err := http.Post(url_.String(), "application/json", &body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("Error pushing build state. Code %d", resp.StatusCode)
	}
	return nil
}

func (u *updater) SetLogs(r *common.Repo, c *common.Commit, b *common.Build, rc io.ReadCloser) error {
	url_, err := url.Parse(u.addr)
	if err != nil {
		return err
	}

	url_.Path = fmt.Sprintf("/api/queue/push/%s/%v/%v/logs", r.FullName, c.Sequence, b.Sequence)
	resp, err := http.Post(url_.String(), "application/json", rc)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("Error pushing build logs. Code %d", resp.StatusCode)
	}
	return nil
}
