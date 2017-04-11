package rpc

import "time"

// Option configures a client option.
type Option func(*Client)

// WithBackoff configures the backoff duration when attempting
// to re-connect to a server.
func WithBackoff(d time.Duration) Option {
	return func(c *Client) {
		c.backoff = d
	}
}

// WithRetryLimit configures the maximum number of retries when
// connecting to the server.
func WithRetryLimit(i int) Option {
	return func(c *Client) {
		c.retry = i
	}
}

// WithToken configures the client authorization token.
func WithToken(t string) Option {
	return func(c *Client) {
		c.token = t
	}
}

// WithHeader configures the client header.
func WithHeader(key, value string) Option {
	return func(c *Client) {
		c.headers[key] = []string{value}
	}
}
