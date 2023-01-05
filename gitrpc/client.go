// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"github.com/harness/gitness/gitrpc/rpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn               *grpc.ClientConn
	repoService        rpc.RepositoryServiceClient
	refService         rpc.ReferenceServiceClient
	httpService        rpc.SmartHTTPServiceClient
	commitFilesService rpc.CommitFilesServiceClient
	diffService        rpc.DiffServiceClient
}

func New(remoteAddr string) (*Client, error) {
	// create interceptors
	logIntc := NewClientLogInterceptor()

	// preparate all grpc options
	grpcOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(
			logIntc.UnaryClientInterceptor(),
		),
		grpc.WithChainStreamInterceptor(
			logIntc.StreamClientInterceptor(),
		),
	}

	conn, err := grpc.Dial(remoteAddr, grpcOpts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:               conn,
		repoService:        rpc.NewRepositoryServiceClient(conn),
		refService:         rpc.NewReferenceServiceClient(conn),
		httpService:        rpc.NewSmartHTTPServiceClient(conn),
		commitFilesService: rpc.NewCommitFilesServiceClient(conn),
		diffService:        rpc.NewDiffServiceClient(conn),
	}, nil
}
