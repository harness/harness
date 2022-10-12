// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"fmt"
	"net"

	"code.gitea.io/gitea/modules/setting"
	"github.com/harness/gitness/internal/gitrpc/rpc"
	"google.golang.org/grpc"
)

type Server struct {
	*grpc.Server
	Port int
}

func NewServer(port int) (*Server, error) {
	setting.Git.HomePath = "home"
	adapter, err := newGitea()
	if err != nil {
		return nil, err
	}
	s := grpc.NewServer()
	store := newLocalStore()
	rpc.RegisterRepositoryServiceServer(s, &repositoryService{adapter: adapter, store: store})
	rpc.RegisterUploadServiceServer(s, newUploadService(adapter, store))
	return &Server{
		Server: s,
		Port:   port,
	}, nil
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		return err
	}
	return s.Server.Serve(lis)
}

func (s *Server) Stop() error {
	s.Server.GracefulStop()
	return nil
}
