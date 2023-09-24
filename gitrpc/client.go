// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gitrpc

import (
	"fmt"
	"time"

	"github.com/harness/gitness/gitrpc/rpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn               *grpc.ClientConn
	repoService        rpc.RepositoryServiceClient
	refService         rpc.ReferenceServiceClient
	httpService        rpc.SmartHTTPServiceClient
	commitFilesService rpc.CommitFilesServiceClient
	diffService        rpc.DiffServiceClient
	mergeService       rpc.MergeServiceClient
	blameService       rpc.BlameServiceClient
	pushService        rpc.PushServiceClient
}

func New(config Config) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("provided config is invalid: %w", err)
	}

	// create interceptors
	logIntc := NewClientLogInterceptor()

	// preparate all grpc options
	grpcOpts := []grpc.DialOption{
		grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"loadBalancingPolicy":"%s"}`, config.LoadBalancingPolicy)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(
			logIntc.UnaryClientInterceptor(),
		),
		grpc.WithChainStreamInterceptor(
			logIntc.StreamClientInterceptor(),
		),
		grpc.WithConnectParams(
			grpc.ConnectParams{
				// This config optimizes for connection recovery instead of load reduction.
				// NOTE: we only expect limited number of internal clients, thus low number of connections.
				Backoff: backoff.Config{
					BaseDelay:  100 * time.Millisecond,
					Multiplier: 1.6, // same as default
					Jitter:     0.2, // same as default
					MaxDelay:   time.Second,
				},
			},
		),
	}

	conn, err := grpc.Dial(config.Addr, grpcOpts...)
	if err != nil {
		return nil, err
	}

	return NewWithConn(conn), nil
}

func NewWithConn(conn *grpc.ClientConn) *Client {
	return &Client{
		conn:               conn,
		repoService:        rpc.NewRepositoryServiceClient(conn),
		refService:         rpc.NewReferenceServiceClient(conn),
		httpService:        rpc.NewSmartHTTPServiceClient(conn),
		commitFilesService: rpc.NewCommitFilesServiceClient(conn),
		diffService:        rpc.NewDiffServiceClient(conn),
		mergeService:       rpc.NewMergeServiceClient(conn),
		blameService:       rpc.NewBlameServiceClient(conn),
		pushService:        rpc.NewPushServiceClient(conn),
	}
}
