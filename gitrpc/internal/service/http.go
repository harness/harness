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

package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/harness/gitness/gitrpc/internal/streamio"
	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"

	"code.gitea.io/gitea/modules/git"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var safeGitProtocolHeader = regexp.MustCompile(`^[0-9a-zA-Z]+=[0-9a-zA-Z]+(:[0-9a-zA-Z]+=[0-9a-zA-Z]+)*$`)

type SmartHTTPService struct {
	rpc.UnimplementedSmartHTTPServiceServer
	adapter   GitAdapter
	reposRoot string
}

func NewHTTPService(adapter GitAdapter, reposRoot string) (*SmartHTTPService, error) {
	return &SmartHTTPService{
		adapter:   adapter,
		reposRoot: reposRoot,
	}, nil
}

func (s *SmartHTTPService) InfoRefs(
	request *rpc.InfoRefsRequest,
	stream rpc.SmartHTTPService_InfoRefsServer,
) error {
	ctx := stream.Context()
	base := request.GetBase()
	if base == nil {
		return types.ErrBaseCannotBeEmpty
	}

	// NOTE: Don't include os.Environ() as we don't have control over it - define everything explicitly
	environ := []string{}
	if request.GitProtocol != "" {
		environ = append(environ, "GIT_PROTOCOL="+request.GitProtocol)
	}

	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	w := streamio.NewWriter(func(p []byte) error {
		return stream.Send(&rpc.InfoRefsResponse{Data: p})
	})

	cmd := &bytes.Buffer{}
	if err := git.NewCommand(ctx, request.GetService(), "--stateless-rpc", "--advertise-refs", ".").
		Run(&git.RunOpts{
			Env:    environ,
			Dir:    repoPath,
			Stdout: cmd,
		}); err != nil {
		return status.Errorf(codes.Internal, "InfoRefsUploadPack: cmd: %v", err)
	}
	if _, err := w.Write(packetWrite("# service=git-" + request.GetService() + "\n")); err != nil {
		return status.Errorf(codes.Internal, "InfoRefsUploadPack: pktLine: %v", err)
	}

	if _, err := w.Write([]byte("0000")); err != nil {
		return status.Errorf(codes.Internal, "InfoRefsUploadPack: flush: %v", err)
	}

	if _, err := io.Copy(w, cmd); err != nil {
		return status.Errorf(codes.Internal, "InfoRefsUploadPack: %v", err)
	}
	return nil
}

func (s *SmartHTTPService) ServicePack(stream rpc.SmartHTTPService_ServicePackServer) error {
	ctx := stream.Context()
	// Get basic repo data
	request, err := stream.Recv()
	if err != nil {
		return err
	}
	// if client sends data as []byte raise error, needs reader
	if request.GetData() != nil {
		return status.Errorf(codes.InvalidArgument, "ServicePack(): non-empty Data")
	}

	// ensure we have the correct base type that matches the services to be triggered
	var repoUID string
	switch request.GetService() {
	case rpc.ServiceUploadPack:
		if request.GetReadBase() == nil {
			return status.Errorf(codes.InvalidArgument, "ServicePack(): read base is missing for upload-pack")
		}
		repoUID = request.GetReadBase().GetRepoUid()
	case rpc.ServiceReceivePack:
		if request.GetWriteBase() == nil {
			return status.Errorf(codes.InvalidArgument, "ServicePack(): write base is missing for receive-pack")
		}
		repoUID = request.GetWriteBase().GetRepoUid()
	default:
		return status.Errorf(codes.InvalidArgument, "ServicePack(): unsupported service '%s'", request.GetService())
	}

	repoPath := getFullPathForRepo(s.reposRoot, repoUID)

	stdin := streamio.NewReader(func() ([]byte, error) {
		resp, streamErr := stream.Recv()
		return resp.GetData(), streamErr
	})

	stdout := streamio.NewWriter(func(p []byte) error {
		return stream.Send(&rpc.ServicePackResponse{Data: p})
	})

	return serviceRPC(ctx, stdin, stdout, request, repoPath)
}

func serviceRPC(ctx context.Context, stdin io.Reader, stdout io.Writer,
	request *rpc.ServicePackRequest, dir string) error {
	protocol := request.GetGitProtocol()
	service := request.GetService()

	// NOTE: Don't include os.Environ() as we don't have control over it - define everything explicitly
	environ := []string{}
	if request.GetWriteBase() != nil {
		// in case of a write operation inject the provided environment variables
		environ = CreateEnvironmentForPush(ctx, request.GetWriteBase())
	}
	// set this for allow pre-receive and post-receive execute
	environ = append(environ, "SSH_ORIGINAL_COMMAND="+service)

	if protocol != "" && safeGitProtocolHeader.MatchString(protocol) {
		environ = append(environ, "GIT_PROTOCOL="+protocol)
	}
	var (
		stderr bytes.Buffer
	)
	cmd := git.NewCommand(ctx, service, "--stateless-rpc", dir)
	cmd.SetDescription(fmt.Sprintf("%s %s %s [repo_path: %s]", git.GitExecutable, service, "--stateless-rpc", dir))
	err := cmd.Run(&git.RunOpts{
		Dir:               dir,
		Env:               environ,
		Stdout:            stdout,
		Stdin:             stdin,
		Stderr:            &stderr,
		UseContextTimeout: true,
	})
	if err != nil && err.Error() != "signal: killed" {
		log.Ctx(ctx).Err(err).Msgf("Fail to serve RPC(%s) in %s: %v - %s", service, dir, err, stderr.String())
	}

	return err
}

func packetWrite(str string) []byte {
	s := strconv.FormatInt(int64(len(str)+4), 16)
	if len(s)%4 != 0 {
		s = strings.Repeat("0", 4-len(s)%4) + s
	}
	return []byte(s + str)
}
