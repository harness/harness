// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/harness/gitness/gitrpc/internal/middleware"
	"github.com/harness/gitness/gitrpc/internal/service"
	"github.com/harness/gitness/gitrpc/internal/storage"
	"github.com/harness/gitness/gitrpc/rpc"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

const (
	repoSubdirName           = "repos"
	ReposGraveyardSubdirName = "cleanup"
)

type GRPCServer struct {
	*grpc.Server
	Bind string
}

func NewServer(config Config, adapter service.GitAdapter) (*GRPCServer, error) {
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
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionAge:      config.MaxConnAge,
			MaxConnectionAgeGrace: config.MaxConnAgeGrace,
		}),
	)
	store := storage.NewLocalStore()
	// create a temp dir for deleted repositories
	// this dir should get cleaned up peridocally if it's not empty
	reposGraveyard := filepath.Join(config.GitRoot, ReposGraveyardSubdirName)
	if _, errdir := os.Stat(reposGraveyard); os.IsNotExist(errdir) {
		if errdir = os.MkdirAll(reposGraveyard, 0o700); errdir != nil {
			return nil, errdir
		}
	}
	// initialize services
	repoService, err := service.NewRepositoryService(adapter, store, reposRoot, config.TmpDir,
		config.GitHookPath, reposGraveyard)
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
	blameService := service.NewBlameService(adapter, reposRoot)
	pushService := service.NewPushService(adapter, reposRoot)

	// register services
	rpc.RegisterRepositoryServiceServer(s, repoService)
	rpc.RegisterReferenceServiceServer(s, refService)
	rpc.RegisterSmartHTTPServiceServer(s, httpService)
	rpc.RegisterCommitFilesServiceServer(s, commitFilesService)
	rpc.RegisterDiffServiceServer(s, diffService)
	rpc.RegisterMergeServiceServer(s, mergeService)
	rpc.RegisterBlameServiceServer(s, blameService)
	rpc.RegisterPushServiceServer(s, pushService)

	return &GRPCServer{
		Server: s,
		Bind:   config.Bind,
	}, nil
}

func (s *GRPCServer) Start() error {
	lis, err := net.Listen("tcp", s.Bind)
	if err != nil {
		return err
	}
	return s.Server.Serve(lis)
}

func (s *GRPCServer) Stop() error {
	s.Server.GracefulStop()
	return nil
}
