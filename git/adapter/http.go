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

package adapter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/harness/gitness/errors"

	"code.gitea.io/gitea/modules/git"
	"github.com/rs/zerolog/log"
)

func (a Adapter) InfoRefs(
	ctx context.Context,
	repoPath string,
	service string,
	w io.Writer,
	env ...string,
) error {
	cmd := &bytes.Buffer{}
	if err := git.NewCommand(ctx, service, "--stateless-rpc", "--advertise-refs", ".").
		Run(&git.RunOpts{
			Env:    env,
			Dir:    repoPath,
			Stdout: cmd,
		}); err != nil {
		return errors.Internal(err, "InfoRefs service %s failed", service)
	}
	if _, err := w.Write(packetWrite("# service=git-" + service + "\n")); err != nil {
		return errors.Internal(err, "failed to write pktLine in InfoRefs %s service", service)
	}

	if _, err := w.Write([]byte("0000")); err != nil {
		return errors.Internal(err, "failed to flush data in InfoRefs %s service", service)
	}

	if _, err := io.Copy(w, cmd); err != nil {
		return errors.Internal(err, "streaming InfoRefs %s service failed", service)
	}
	return nil
}

func (a Adapter) ServicePack(
	ctx context.Context,
	repoPath string,
	service string,
	stdin io.Reader,
	stdout io.Writer,
	env ...string,
) error {
	// set this for allow pre-receive and post-receive execute
	env = append(env, "SSH_ORIGINAL_COMMAND="+service)

	var (
		stderr bytes.Buffer
	)
	cmd := git.NewCommand(ctx, service, "--stateless-rpc", repoPath)
	cmd.SetDescription(fmt.Sprintf("%s %s %s [repo_path: %s]", git.GitExecutable, service, "--stateless-rpc", repoPath))
	err := cmd.Run(&git.RunOpts{
		Dir:               repoPath,
		Env:               env,
		Stdout:            stdout,
		Stdin:             stdin,
		Stderr:            &stderr,
		UseContextTimeout: true,
	})
	if err != nil && err.Error() != "signal: killed" {
		log.Ctx(ctx).Err(err).Msgf("Fail to serve RPC(%s) in %s: %v - %s", service, repoPath, err, stderr.String())
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
