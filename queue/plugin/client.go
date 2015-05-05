package plugin

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/drone/drone/queue"
)

type Client struct {
	url   string
	token string
}

func New(url, token string) *Client {
	return &Client{url, token}
}

// Publish makes an http request to the remote queue
// to insert work at the tail.
func (c *Client) Publish(work *queue.Work) error {
	return c.send("POST", "/queue", work, nil)
}

// Remove makes an http request to the remote queue to
// remove the specified work item.
func (c *Client) Remove(work *queue.Work) error {
	return c.send("DELETE", "/queue", work, nil)
}

// Pull makes an http request to the remote queue to
// retrieve work. This initiates a long poll and will
// block until complete.
func (c *Client) Pull() *queue.Work {
	out := &queue.Work{}
	err := c.send("POST", "/queue/pull", nil, out)
	if err != nil {
		// TODO handle error
	}
	return out
}

// Pull makes an http request to the remote queue to
// retrieve work. This initiates a long poll and will
// block until complete.
func (c *Client) PullClose(cn queue.CloseNotifier) *queue.Work {
	out := &queue.Work{}
	err := c.send("POST", "/queue/pull", nil, out)
	if err != nil {
		// TODO handle error
	}
	return out
}

// Ack makes an http request to the remote queue
// to acknowledge an item in the queue was processed.
func (c *Client) Ack(work *queue.Work) error {
	return c.send("POST", "/queue/ack", nil, nil)
}

// Items makes an http request to the remote queue
// to fetch a list of all work.
func (c *Client) Items() []*queue.Work {
	out := []*queue.Work{}
	err := c.send("GET", "/queue/items", nil, &out)
	if err != nil {
		// TODO handle error
	}
	return out
}

// send is a helper function that makes an authenticated
// request to the remote http plugin.
func (c *Client) send(method, path string, in interface{}, out interface{}) error {
	url_, err := url.Parse(c.url + path)
	if err != nil {
		return err
	}

	var buf io.ReadWriter
	if in != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(in)
		if err != nil {
			return err
		}
	}

	req, err := http.NewRequest(method, url_.String(), buf)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+c.token)
	req.Header.Add("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// In order to implement PullClose() we'll need to use a custom transport:
//
// tr := &http.Transport{}
// client := &http.Client{Transport: tr}
// c := make(chan error, 1)
// go func() { c <- f(client.Do(req)) }()
// select {
// case <-ctx.Done():
//     tr.CancelRequest(req)
//     <-c // Wait for f to return.
//     return ctx.Err()
// case err := <-c:
//     return err
// }
