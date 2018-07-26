// Copyright 2018 Drone.IO Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"

	"github.com/cncd/pipeline/pipeline"
	"github.com/cncd/pipeline/pipeline/backend"
	"github.com/cncd/pipeline/pipeline/backend/docker"
	"github.com/cncd/pipeline/pipeline/multipart"
	"github.com/cncd/pipeline/pipeline/rpc"

	"github.com/drone/signal"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tevino/abool"
	"github.com/urfave/cli"
	oldcontext "golang.org/x/net/context"
)

func loop(c *cli.Context) error {
	filter := rpc.Filter{
		Labels: map[string]string{
			"platform": c.String("platform"),
		},
		Expr: c.String("filter"),
	}

	hostname := c.String("hostname")
	if len(hostname) == 0 {
		hostname, _ = os.Hostname()
	}

	if c.BoolT("debug") {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	}

	if c.Bool("pretty") {
		log.Logger = log.Output(
			zerolog.ConsoleWriter{
				Out:     os.Stderr,
				NoColor: c.BoolT("nocolor"),
			},
		)
	}

	counter.Polling = c.Int("max-procs")
	counter.Running = 0

	if c.BoolT("healthcheck") {
		go http.ListenAndServe(":3000", nil)
	}

	// TODO pass version information to grpc server
	// TODO authenticate to grpc server

	// grpc.Dial(target, ))

	conn, err := grpc.Dial(
		c.String("server"),
		grpc.WithInsecure(),
		grpc.WithPerRPCCredentials(&credentials{
			username: c.String("username"),
			password: c.String("password"),
		}),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:    c.Duration("keepalive-time"),
			Timeout: c.Duration("keepalive-timeout"),
		}),
	)

	if err != nil {
		return err
	}
	defer conn.Close()

	client := rpc.NewGrpcClient(conn)

	sigterm := abool.New()
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.Pairs("hostname", hostname),
	)
	ctx = signal.WithContextFunc(ctx, func() {
		println("ctrl+c received, terminating process")
		sigterm.Set()
	})

	var wg sync.WaitGroup
	parallel := c.Int("max-procs")
	wg.Add(parallel)

	for i := 0; i < parallel; i++ {
		go func() {
			defer wg.Done()
			for {
				if sigterm.IsSet() {
					return
				}
				r := runner{
					client:   client,
					filter:   filter,
					hostname: hostname,
				}
				if err := r.run(ctx); err != nil {
					log.Error().Err(err).Msg("pipeline done with error")
					return
				}
			}
		}()
	}

	wg.Wait()
	return nil
}

// NOTE we need to limit the size of the logs and files that we upload.
// The maximum grpc payload size is 4194304. So until we implement streaming
// for uploads, we need to set these limits below the maximum.
const (
	maxLogsUpload = 4000000 // this is per step
	maxFileUpload = 1000000
)

type runner struct {
	client   rpc.Peer
	filter   rpc.Filter
	hostname string
}

func (r *runner) run(ctx context.Context) error {
	log.Debug().
		Msg("request next execution")

	meta, _ := metadata.FromOutgoingContext(ctx)
	ctxmeta := metadata.NewOutgoingContext(context.Background(), meta)

	// get the next job from the queue
	work, err := r.client.Next(ctx, r.filter)
	if err != nil {
		return err
	}
	if work == nil {
		return nil
	}

	timeout := time.Hour
	if minutes := work.Timeout; minutes != 0 {
		timeout = time.Duration(minutes) * time.Minute
	}

	counter.Add(
		work.ID,
		timeout,
		extractRepositoryName(work.Config), // hack
		extractBuildNumber(work.Config),    // hack
	)
	defer counter.Done(work.ID)

	logger := log.With().
		Str("repo", extractRepositoryName(work.Config)). // hack
		Str("build", extractBuildNumber(work.Config)).   // hack
		Str("id", work.ID).
		Logger()

	logger.Debug().
		Msg("received execution")

	// new docker engine
	engine, err := docker.NewEnv()
	if err != nil {
		logger.Error().
			Err(err).
			Msg("cannot create docker client")

		return err
	}

	ctx, cancel := context.WithTimeout(ctxmeta, timeout)
	defer cancel()

	cancelled := abool.New()
	go func() {
		logger.Debug().
			Msg("listen for cancel signal")

		if werr := r.client.Wait(ctx, work.ID); werr != nil {
			cancelled.SetTo(true)
			logger.Warn().
				Err(werr).
				Msg("cancel signal received")

			cancel()
		} else {
			logger.Debug().
				Msg("stop listening for cancel signal")
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Debug().
					Msg("pipeline done")

				return
			case <-time.After(time.Minute):
				logger.Debug().
					Msg("pipeline lease renewed")

				r.client.Extend(ctx, work.ID)
			}
		}
	}()

	state := rpc.State{}
	state.Started = time.Now().Unix()

	err = r.client.Init(ctxmeta, work.ID, state)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("pipeline initialization failed")
	}

	var uploads sync.WaitGroup
	defaultLogger := pipeline.LogFunc(func(proc *backend.Step, rc multipart.Reader) error {

		loglogger := logger.With().
			Str("image", proc.Image).
			Str("stage", proc.Alias).
			Logger()

		part, rerr := rc.NextPart()
		if rerr != nil {
			return rerr
		}
		uploads.Add(1)

		var secrets []string
		for _, secret := range work.Config.Secrets {
			if secret.Mask {
				secrets = append(secrets, secret.Value)
			}
		}

		loglogger.Debug().Msg("log stream opened")

		logstream := rpc.NewLineWriter(r.client, work.ID, proc.Alias, secrets...)
		io.Copy(logstream, part)

		loglogger.Debug().Msg("log stream copied")

		// maxLogsUpload is now more accurate
		// We want the end of the logs, not the beginning
		logLines := logstream.Lines()
		fileData, _ := json.Marshal(logLines)
		fileBuffer := bytes.NewBuffer(fileData)
		suffixBytes := []byte("\\\"},")
		var firstLine uint = 0
		for ; fileBuffer.Len() > maxLogsUpload; firstLine++ {
			// read bytes until we find '"},' in the encoded JSON
			for c, _ := fileBuffer.ReadBytes(','); bytes.HasSuffix(c, suffixBytes) || !bytes.HasSuffix(c, suffixBytes[1:]); {
				c, _ = fileBuffer.ReadBytes(',')
			}
		}
		fileData, _ = json.Marshal(logLines[firstLine:])
		for ; len(fileData) > maxLogsUpload; firstLine++ {
			fileData, _ = json.Marshal(logLines[firstLine:])
		}

		file := &rpc.File{}
		file.Mime = "application/json+logs"
		file.Proc = proc.Alias
		file.Name = "logs.json"
		file.Data = fileData
		file.Size = len(file.Data)
		file.Time = time.Now().Unix()

		loglogger.Debug().
			Msg("log stream uploading")

		if serr := r.client.Upload(ctxmeta, work.ID, file); serr != nil {
			loglogger.Error().
				Err(serr).
				Msg("log stream upload error")
		}

		loglogger.Debug().
			Msg("log stream upload complete")

		defer func() {
			loglogger.Debug().
				Msg("log stream closed")

			uploads.Done()
		}()

		part, rerr = rc.NextPart()
		if rerr != nil {
			return nil
		}
		// TODO should be configurable
		limitedPart := io.LimitReader(part, maxFileUpload)
		file = &rpc.File{}
		file.Mime = part.Header().Get("Content-Type")
		file.Proc = proc.Alias
		file.Name = part.FileName()
		file.Data, _ = ioutil.ReadAll(limitedPart)
		file.Size = len(file.Data)
		file.Time = time.Now().Unix()
		file.Meta = map[string]string{}

		for key, value := range part.Header() {
			file.Meta[key] = value[0]
		}

		loglogger.Debug().
			Str("file", file.Name).
			Str("mime", file.Mime).
			Msg("file stream uploading")

		if serr := r.client.Upload(ctxmeta, work.ID, file); serr != nil {
			loglogger.Error().
				Err(serr).
				Str("file", file.Name).
				Str("mime", file.Mime).
				Msg("file stream upload error")
		}

		loglogger.Debug().
			Str("file", file.Name).
			Str("mime", file.Mime).
			Msg("file stream upload complete")
		return nil
	})

	defaultTracer := pipeline.TraceFunc(func(state *pipeline.State) error {
		proclogger := logger.With().
			Str("image", state.Pipeline.Step.Image).
			Str("stage", state.Pipeline.Step.Alias).
			Int("exit_code", state.Process.ExitCode).
			Bool("exited", state.Process.Exited).
			Logger()

		procState := rpc.State{
			Proc:     state.Pipeline.Step.Alias,
			Exited:   state.Process.Exited,
			ExitCode: state.Process.ExitCode,
			Started:  time.Now().Unix(), // TODO do not do this
			Finished: time.Now().Unix(),
		}
		defer func() {
			proclogger.Debug().
				Msg("update step status")

			if uerr := r.client.Update(ctxmeta, work.ID, procState); uerr != nil {
				proclogger.Debug().
					Err(uerr).
					Msg("update step status error")
			}

			proclogger.Debug().
				Msg("update step status complete")
		}()
		if state.Process.Exited {
			return nil
		}
		if state.Pipeline.Step.Environment == nil {
			state.Pipeline.Step.Environment = map[string]string{}
		}

		state.Pipeline.Step.Environment["DRONE_MACHINE"] = r.hostname
		state.Pipeline.Step.Environment["CI_BUILD_STATUS"] = "success"
		state.Pipeline.Step.Environment["CI_BUILD_STARTED"] = strconv.FormatInt(state.Pipeline.Time, 10)
		state.Pipeline.Step.Environment["CI_BUILD_FINISHED"] = strconv.FormatInt(time.Now().Unix(), 10)
		state.Pipeline.Step.Environment["DRONE_BUILD_STATUS"] = "success"
		state.Pipeline.Step.Environment["DRONE_BUILD_STARTED"] = strconv.FormatInt(state.Pipeline.Time, 10)
		state.Pipeline.Step.Environment["DRONE_BUILD_FINISHED"] = strconv.FormatInt(time.Now().Unix(), 10)

		state.Pipeline.Step.Environment["CI_JOB_STATUS"] = "success"
		state.Pipeline.Step.Environment["CI_JOB_STARTED"] = strconv.FormatInt(state.Pipeline.Time, 10)
		state.Pipeline.Step.Environment["CI_JOB_FINISHED"] = strconv.FormatInt(time.Now().Unix(), 10)
		state.Pipeline.Step.Environment["DRONE_JOB_STATUS"] = "success"
		state.Pipeline.Step.Environment["DRONE_JOB_STARTED"] = strconv.FormatInt(state.Pipeline.Time, 10)
		state.Pipeline.Step.Environment["DRONE_JOB_FINISHED"] = strconv.FormatInt(time.Now().Unix(), 10)

		if state.Pipeline.Error != nil {
			state.Pipeline.Step.Environment["CI_BUILD_STATUS"] = "failure"
			state.Pipeline.Step.Environment["CI_JOB_STATUS"] = "failure"
			state.Pipeline.Step.Environment["DRONE_BUILD_STATUS"] = "failure"
			state.Pipeline.Step.Environment["DRONE_JOB_STATUS"] = "failure"
		}
		return nil
	})

	err = pipeline.New(work.Config,
		pipeline.WithContext(ctx),
		pipeline.WithLogger(defaultLogger),
		pipeline.WithTracer(defaultTracer),
		pipeline.WithEngine(engine),
	).Run()

	state.Finished = time.Now().Unix()
	state.Exited = true
	if err != nil {
		switch xerr := err.(type) {
		case *pipeline.ExitError:
			state.ExitCode = xerr.Code
		default:
			state.ExitCode = 1
			state.Error = err.Error()
		}
		if cancelled.IsSet() {
			state.ExitCode = 137
		}
	}

	logger.Debug().
		Str("error", state.Error).
		Int("exit_code", state.ExitCode).
		Msg("pipeline complete")

	logger.Debug().
		Msg("uploading logs")

	uploads.Wait()

	logger.Debug().
		Msg("uploading logs complete")

	logger.Debug().
		Str("error", state.Error).
		Int("exit_code", state.ExitCode).
		Msg("updating pipeline status")

	err = r.client.Done(ctxmeta, work.ID, state)
	if err != nil {
		logger.Error().Err(err).
			Msg("updating pipeline status failed")
	} else {
		logger.Debug().
			Msg("updating pipeline status complete")
	}

	return nil
}

type credentials struct {
	username string
	password string
}

func (c *credentials) GetRequestMetadata(oldcontext.Context, ...string) (map[string]string, error) {
	return map[string]string{
		"username": c.username,
		"password": c.password,
	}, nil
}

func (c *credentials) RequireTransportSecurity() bool {
	return false
}

// extract repository name from the configuration
func extractRepositoryName(config *backend.Config) string {
	return config.Stages[0].Steps[0].Environment["DRONE_REPO"]
}

// extract build number from the configuration
func extractBuildNumber(config *backend.Config) string {
	return config.Stages[0].Steps[0].Environment["DRONE_BUILD_NUMBER"]
}
