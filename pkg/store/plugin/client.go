package plugin

import (
	"net"
	"net/rpc"
)

type Client struct {
	*rpc.Client
}

// New returns a new, remote datastore backend that connects
// via tcp and exchanges data using Go's RPC mechanism.
func New(address string) (*Client, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	client := &Client{
		rpc.NewClient(conn),
	}
	return client, nil
}
