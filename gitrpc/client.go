// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"github.com/harness/gitness/gitrpc/rpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config represents the config for the gitrpc client.
type Config struct {
	Bind string
}

type Client struct {
	conn        *grpc.ClientConn
	repoService rpc.RepositoryServiceClient
	httpService rpc.SmartHTTPServiceClient
}

func New(remoteAddr string) (*Client, error) {
	conn, err := grpc.Dial(remoteAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:        conn,
		repoService: rpc.NewRepositoryServiceClient(conn),
		httpService: rpc.NewSmartHTTPServiceClient(conn),
	}, nil
}
