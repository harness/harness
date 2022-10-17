// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"net"

	"code.gitea.io/gitea/modules/setting"
	"github.com/harness/gitness/internal/gitrpc/rpc"
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
	adapter, err := newGiteaAdapter()
	if err != nil {
		return nil, err
	}
	s := grpc.NewServer()
	store := newLocalStore()
	repoService, err := newRepositoryService(adapter, store, gitRoot)
	if err != nil {
		return nil, err
	}
	rpc.RegisterRepositoryServiceServer(s, repoService)
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
