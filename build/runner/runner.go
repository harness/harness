// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package runner

import (
	"os"

	"github.com/google/uuid"
	"github.com/harness/gitness/build/manager"
	"github.com/harness/gitness/types"

	"github.com/drone-runners/drone-runner-docker/engine"
	"github.com/drone-runners/drone-runner-docker/engine/compiler"
	"github.com/drone-runners/drone-runner-docker/engine/linter"
	"github.com/drone-runners/drone-runner-docker/engine/resource"
	"github.com/drone/drone-go/drone"
	runnerclient "github.com/drone/runner-go/client"
	"github.com/drone/runner-go/environ/provider"
	"github.com/drone/runner-go/pipeline/reporter/history"
	"github.com/drone/runner-go/pipeline/reporter/remote"
	"github.com/drone/runner-go/pipeline/runtime"
	"github.com/drone/runner-go/pipeline/uploader"
	"github.com/drone/runner-go/registry"
	"github.com/drone/runner-go/secret"
)

func NewExecutionRunner(
	config *types.Config,
	client runnerclient.Client,
	m manager.ExecutionManager,
) (*runtime.Runner, error) {
	compiler := &compiler.Compiler{
		Environ:  provider.Static(map[string]string{}),
		Registry: registry.Static([]*drone.Registry{}),
		Secret:   secret.Encrypted(),
	}

	var host string
	host, err := os.Hostname()
	if err != nil {
		host = uuid.New().String()
	}
	remote := remote.New(client)
	upload := uploader.New(client)
	tracer := history.New(remote)
	engine, err := engine.NewEnv(engine.Opts{})
	if err != nil {
		return nil, err
	}
	// TODO: Using the same parallel workers as the max concurrent step limit,
	// this can be made configurable if needed later.
	exec := runtime.NewExecer(tracer, remote, upload,
		engine, int64(config.CI.ParallelWorkers))
	runner := &runtime.Runner{
		Machine:  host, // TODO: Check whether this needs to be configurable
		Client:   client,
		Reporter: tracer,
		Lookup:   resource.Lookup,
		Lint:     linter.New().Lint,
		Compiler: compiler,
		Exec:     exec.Exec,
	}
	return runner, nil
}
