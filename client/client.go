package client

import (
	"io"

	"github.com/drone/drone/model"
	"github.com/drone/drone/queue"
)

// Client is used to communicate with a Drone server.
type Client interface {
	// Sign returns a cryptographic signature for the input string.
	Sign(string, string, []byte) ([]byte, error)

	// SecretPost create or updates a repository secret.
	SecretPost(string, string, *model.Secret) error

	// SecretDel deletes a named repository secret.
	SecretDel(string, string, string) error

	// Pull pulls work from the server queue.
	Pull(os, arch string) (*queue.Work, error)

	// Push pushes an update to the server.
	Push(*queue.Work) error

	// Stream streams the build logs to the server.
	Stream(int64, io.ReadCloser) error

	// Wait waits for the job to the complete.
	Wait(int64) *Wait
}
