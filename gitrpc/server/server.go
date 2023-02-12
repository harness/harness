// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package server

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/harness/gitness/gitrpc/internal/gitea"
	"github.com/harness/gitness/gitrpc/internal/middleware"
	"github.com/harness/gitness/gitrpc/internal/service"
	"github.com/harness/gitness/gitrpc/internal/storage"
	"github.com/harness/gitness/gitrpc/rpc"

	"code.gitea.io/gitea/modules/setting"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"google.golang.org/grpc"
)

const (
	repoSubdirName = "repos"
)

type Server struct {
	*grpc.Server
	Bind string
}

func NewServer(config Config) (*Server, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration is invalid: %w", err)
	}
	// Create repos folder
	reposRoot := filepath.Join(config.GitRoot, repoSubdirName)
	if _, err := os.Stat(reposRoot); errors.Is(err, os.ErrNotExist) {
		if err = os.MkdirAll(reposRoot, 0o700); err != nil {
			return nil, err
		}
	}

	// TODO: should be subdir of gitRoot? What is it being used for?
	setting.Git.HomePath = "home"
	adapter, err := gitea.New()
	if err != nil {
		return nil, err
	}

	// interceptors
	errIntc := middleware.NewErrInterceptor()
	logIntc := middleware.NewLogInterceptor()

	s := grpc.NewServer(
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_recovery.UnaryServerInterceptor(),
			logIntc.UnaryInterceptor(),
			errIntc.UnaryInterceptor(),
		)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_recovery.StreamServerInterceptor(),
			logIntc.StreamInterceptor(),
			errIntc.StreamInterceptor(),
		)),
	)
	store := storage.NewLocalStore()

	// initialize services
	repoService, err := service.NewRepositoryService(adapter, store, reposRoot, config.GitHookPath)
	if err != nil {
		return nil, err
	}
	refService, err := service.NewReferenceService(adapter, reposRoot, config.TmpDir)
	if err != nil {
		return nil, err
	}
	httpService, err := service.NewHTTPService(adapter, reposRoot)
	if err != nil {
		return nil, err
	}
	commitFilesService, err := service.NewCommitFilesService(adapter, reposRoot, config.TmpDir)
	if err != nil {
		return nil, err
	}
	diffService, err := service.NewDiffService(adapter, reposRoot, config.TmpDir)
	if err != nil {
		return nil, err
	}
	mergeService, err := service.NewMergeService(adapter, reposRoot, config.TmpDir)
	if err != nil {
		return nil, err
	}

	// register services
	rpc.RegisterRepositoryServiceServer(s, repoService)
	rpc.RegisterReferenceServiceServer(s, refService)
	rpc.RegisterSmartHTTPServiceServer(s, httpService)
	rpc.RegisterCommitFilesServiceServer(s, commitFilesService)
	rpc.RegisterDiffServiceServer(s, diffService)
	rpc.RegisterMergeServiceServer(s, mergeService)

	return &Server{
		Server: s,
		Bind:   config.Bind,
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
