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

package api

import (
	"bytes"
	"context"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

var safeGitProtocolHeader = regexp.MustCompile(`^[0-9a-zA-Z]+=[0-9a-zA-Z]+(:[0-9a-zA-Z]+=[0-9a-zA-Z]+)*$`)

func (g *Git) InfoRefs(
	ctx context.Context,
	repoPath string,
	service string,
	w io.Writer,
	env ...string,
) error {
	stdout := &bytes.Buffer{}
	cmd := command.New(service,
		command.WithFlag("--stateless-rpc"),
		command.WithFlag("--advertise-refs"),
		command.WithArg("."),
	)
	if err := cmd.Run(ctx,
		command.WithDir(repoPath),
		command.WithStdout(stdout),
		command.WithEnvs(env...),
	); err != nil {
		return errors.Internal(err, "InfoRefs service %s failed", service)
	}
	if _, err := w.Write(packetWrite("# service=git-" + service + "\n")); err != nil {
		return errors.Internal(err, "failed to write pktLine in InfoRefs %s service", service)
	}

	if _, err := w.Write([]byte("0000")); err != nil {
		return errors.Internal(err, "failed to flush data in InfoRefs %s service", service)
	}

	if _, err := io.Copy(w, stdout); err != nil {
		return errors.Internal(err, "streaming InfoRefs %s service failed", service)
	}
	return nil
}

type ServicePackOptions struct {
	Service      enum.GitServiceType
	Timeout      int // seconds
	StatelessRPC bool
	Stdout       io.Writer
	Stdin        io.Reader
	Stderr       io.Writer
	Env          []string
	Protocol     string
}

func (g *Git) ServicePack(
	ctx context.Context,
	repoPath string,
	options ServicePackOptions,
) error {
	cmd := command.New(string(options.Service),
		command.WithArg(repoPath),
		command.WithEnv("SSH_ORIGINAL_COMMAND", string(options.Service)),
	)

	if options.StatelessRPC {
		cmd.Add(command.WithFlag("--stateless-rpc"))
	}

	if options.Protocol != "" && safeGitProtocolHeader.MatchString(options.Protocol) {
		cmd.Add(command.WithEnv("GIT_PROTOCOL", options.Protocol))
	}

	err := cmd.Run(ctx,
		command.WithDir(repoPath),
		command.WithStdout(options.Stdout),
		command.WithStdin(options.Stdin),
		command.WithStderr(options.Stderr),
		command.WithEnvs(options.Env...),
	)
	if err != nil && err.Error() != "signal: killed" {
		log.Ctx(ctx).Err(err).Msgf("Fail to serve RPC(%s) in %s: %v", options.Service, repoPath, err)
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
