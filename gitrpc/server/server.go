// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package server

import (
	"github.com/harness/gitness/gitrpc/internal/gitea"
	"github.com/harness/gitness/gitrpc/internal/service"
	"github.com/harness/gitness/gitrpc/internal/storage"
	"github.com/harness/gitness/gitrpc/rpc"

	"net"

	"code.gitea.io/gitea/modules/setting"
	"google.golang.org/grpc"
)

type Server struct {
	*grpc.Server
	Bind string
}

// TODO: this wiring should be done by wire.
func NewServer(bind string, gitRoot string) (*Server, error) {
	// TODO: should be subdir of gitRoot? What is it being used for?
	setting.Git.HomePath = "home"
	adapter, err := gitea.New()
	if err != nil {
		return nil, err
	}
	s := grpc.NewServer()
	store := storage.NewLocalStore()
	// initialize services
	repoService, err := service.NewRepositoryService(adapter, store, gitRoot)
	if err != nil {
		return nil, err
	}
	httpService, err := service.NewHTTPService(adapter, gitRoot)
	if err != nil {
		return nil, err
	}
	// register services
	rpc.RegisterRepositoryServiceServer(s, repoService)
	rpc.RegisterSmartHTTPServiceServer(s, httpService)

	return &Server{
		Server: s,
		Bind:   bind,
	}, nil
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp", s.Bind)
	if err != nil {
		return err
	}
	return s.Server.Serve(lis)
}

func (s *Server) Stop() error {
	s.Server.GracefulStop()
	return nil
}
