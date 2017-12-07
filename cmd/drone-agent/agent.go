package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/drone/drone-runtime/engine"
	"github.com/drone/drone-runtime/engine/docker"
	"github.com/drone/drone-runtime/engine/plugin"
	"github.com/drone/drone-runtime/runtime"
	"github.com/drone/drone-runtime/runtime/chroot"
	"github.com/drone/drone/server/rpc"

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
		Expr: c.String("drone-filter"),
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

	var err error
	var eng engine.Engine
	if pluginPath := c.String("runtime-plugin"); pluginPath == "" {
		eng, err = docker.NewEnv()
		if err != nil {
			log.Error().
				Err(err).
				Msg("failed to create docker engine")
			return err
		}
	} else {
		eng, err = plugin.Open(pluginPath)
		if err != nil {
			log.Error().
				Err(err).
				Msg("failed to open plugin as engine")
			return err
		}
	}

	counter.Polling = c.Int("max-procs")
	counter.Running = 0

	if c.BoolT("healthcheck") {
		go http.ListenAndServe(":3000", nil)
	}

	conn, err := grpc.Dial(
		c.String("server"),
		grpc.WithInsecure(),
		grpc.WithPerRPCCredentials(&credentials{
			username: c.String("username"),
			password: c.String("password"),
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
					engine:   eng,
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
	maxLogsUpload = 2000000 // this is per step
	maxFileUpload = 1000000
)

type runner struct {
	engine   engine.Engine
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

	hooks := runtime.Hook{}

	//
	// Send the full logs to the server on completion. The log
	// stream is in-memory and volatile so we submit the full
	// logs at the end of the build for guaranteed persistence.
	//

	hooks.GotLogs = func(state *runtime.State, lines []*runtime.Line) error {
		loglogger := logger.With().
			Str("image", state.Step.Image).
			Str("stage", state.Step.Alias).
			Logger()

		file := &rpc.File{}
		file.Mime = "application/json+logs"
		file.Proc = state.Step.Alias
		file.Name = "logs.json"
		file.Data, _ = json.Marshal(lines)
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
		return nil
	}

	//
	// Send each line of logs to the server.
	//

	hooks.GotLine = func(state *runtime.State, line *runtime.Line) error {
		// TODO we do not currently have any sort of log limit here.
		l := &rpc.Line{
			Out:  line.Message,
			Proc: state.Step.Alias,
			Pos:  line.Number,
			// TODO we need to track the start time somewhere
			// Time: int64(time.Since(w.now).Seconds()),
			Type: rpc.LineStdout,
		}
		r.client.Log(context.Background(), work.ID, l)
		return nil
	}

	//
	// Update the server to signal the step is complete
	//

	hooks.AfterEach = func(state *runtime.State) error {
		proclogger := logger.With().
			Str("image", state.Step.Image).
			Str("stage", state.Step.Alias).
			Int("exit_code", state.State.ExitCode).
			Bool("exited", state.State.Exited).
			Logger()

		procState := rpc.State{
			Proc:     state.Step.Alias,
			Exited:   state.State.Exited,
			ExitCode: state.State.ExitCode,
			Started:  time.Now().Unix(), // TODO FIX ME do not do this
			Finished: time.Now().Unix(),
		}

		proclogger.Debug().
			Msg("update step status")

		if uerr := r.client.Update(ctxmeta, work.ID, procState); uerr != nil {
			proclogger.Debug().
				Err(uerr).
				Msg("update step status error")
		}

		proclogger.Debug().
			Msg("update step status complete")
		return nil
	}

	//
	// Update the server to signal the step is starting. Also
	// update the step to set some dynamic runtime environment
	// variables, such as start time and current status.
	//

	hooks.BeforeEach = func(state *runtime.State) error {
		proclogger := logger.With().
			Str("image", state.Step.Image).
			Str("stage", state.Step.Alias).
			Int("exit_code", state.State.ExitCode).
			Bool("exited", state.State.Exited).
			Logger()

		procState := rpc.State{
			Proc:     state.Step.Alias,
			Exited:   state.State.Exited,
			ExitCode: state.State.ExitCode,
			Started:  time.Now().Unix(), // TODO FIX ME do not do this
			Finished: time.Now().Unix(),
		}

		proclogger.Debug().
			Msg("update step status")

		if uerr := r.client.Update(ctxmeta, work.ID, procState); uerr != nil {
			proclogger.Debug().
				Err(uerr).
				Msg("update step status error")
		}

		proclogger.Debug().
			Msg("update step status complete")

		if state.Step.Environment == nil {
			state.Step.Environment = map[string]string{}
		}

		state.Step.Environment["DRONE_MACHINE"] = r.hostname
		state.Step.Environment["CI_BUILD_STATUS"] = "success"
		state.Step.Environment["CI_BUILD_STARTED"] = strconv.FormatInt(state.Runtime.Time, 10)
		state.Step.Environment["CI_BUILD_FINISHED"] = strconv.FormatInt(time.Now().Unix(), 10)
		state.Step.Environment["DRONE_BUILD_STATUS"] = "success"
		state.Step.Environment["DRONE_BUILD_STARTED"] = strconv.FormatInt(state.Runtime.Time, 10)
		state.Step.Environment["DRONE_BUILD_FINISHED"] = strconv.FormatInt(time.Now().Unix(), 10)

		state.Step.Environment["CI_JOB_STATUS"] = "success"
		state.Step.Environment["CI_JOB_STARTED"] = strconv.FormatInt(state.Runtime.Time, 10)
		state.Step.Environment["CI_JOB_FINISHED"] = strconv.FormatInt(time.Now().Unix(), 10)
		state.Step.Environment["DRONE_JOB_STATUS"] = "success"
		state.Step.Environment["DRONE_JOB_STARTED"] = strconv.FormatInt(state.Runtime.Time, 10)
		state.Step.Environment["DRONE_JOB_FINISHED"] = strconv.FormatInt(time.Now().Unix(), 10)

		if state.Runtime.Error != nil {
			state.Step.Environment["CI_BUILD_STATUS"] = "failure"
			state.Step.Environment["CI_JOB_STATUS"] = "failure"
			state.Step.Environment["DRONE_BUILD_STATUS"] = "failure"
			state.Step.Environment["DRONE_JOB_STATUS"] = "failure"
		}

		return nil
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

	fs, _ := chroot.New("/") // TODO this is bad. fix this
	err = runtime.New(
		runtime.WithEngine(r.engine),
		runtime.WithConfig(work.Config),
		runtime.WithFileSystem(fs),
		runtime.WithHooks(&hooks),
	).Run(ctx)

	state.Finished = time.Now().Unix()
	state.Exited = true
	if err != nil {
		switch xerr := err.(type) {
		case *runtime.ExitError:
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
func extractRepositoryName(config *engine.Config) string {
	return config.Stages[0].Steps[0].Environment["DRONE_REPO"]
}

// extract build number from the configuration
func extractBuildNumber(config *engine.Config) string {
	return config.Stages[0].Steps[0].Environment["DRONE_BUILD_NUMBER"]
}
