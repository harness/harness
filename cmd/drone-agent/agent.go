package main

import (
	"context"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/cncd/pipeline/pipeline/backend"
	"github.com/cncd/pipeline/pipeline/rpc"
	"github.com/drone/drone-runtime/engine"
	"github.com/drone/drone-runtime/engine/docker"
	"github.com/drone/drone-runtime/engine/plugin"
	"github.com/drone/drone-runtime/runtime"
	"github.com/drone/drone-runtime/runtime/chroot"
	"github.com/drone/drone-runtime/runtime/term"
	"github.com/drone/signal"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tevino/abool"
	"github.com/urfave/cli"
	oldcontext "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
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
					plugin:   c.String("runtime-plugin"),
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
	client   rpc.Peer
	filter   rpc.Filter
	hostname string
	plugin   string
}

func (r *runner) run(ctx context.Context) error {
	log.Debug().Msg("request next execution")

	meta, _ := metadata.FromOutgoingContext(ctx)
	ctxmeta := metadata.NewOutgoingContext(context.Background(), meta)

	// TODO: Change this to receive the actual new payload
	// get the next job from the queue
	_, err := r.client.Next(ctx, r.filter)
	if err != nil {
		return err
	}

	// TODO: Remove this mocked config
	config := &engine.Config{
		Version: 1,
		Stages: []*engine.Stage{{
			Name:  "stage_0",
			Alias: "stage_0",
			Steps: []*engine.Step{{
				Name:       "step_0",
				Alias:      "step_0",
				Image:      "alpine:3.6",
				WorkingDir: "/",
				Entrypoint: []string{
					"/bin/sh",
					"-c",
				},
				Command: []string{
					"for i in $(seq 1 5); do date && sleep 1; done",
				},
				OnSuccess: true,
			}},
		}},
	}

	timeout := time.Hour
	//if minutes := work.Timeout; minutes != 0 {
	//	timeout = time.Duration(minutes) * time.Minute
	//}

	//counter.Add(
	//	work.ID,
	//	timeout,
	//	extractRepositoryName(work.Config), // hack
	//	extractBuildNumber(work.Config),    // hack
	//)
	//defer counter.Done(work.ID)
	//
	logger := log.With().
		//Str("repo", extractRepositoryName(work.Config)). // hack
		//Str("build", extractBuildNumber(work.Config)).   // hack
		//Str("id", work.ID).
		Logger()

	logger.Debug().Msg("received execution")

	var engine engine.Engine
	if r.plugin == "" {
		engine, err = docker.NewEnv()
		if err != nil {
			logger.Warn().Err(err).Msg("failed to create docker engine")
			return err
		}
	} else {
		engine, err = plugin.Open(r.plugin)
		if err != nil {
			logger.Warn().Err(err).Msg("failed to open plugin as engine")
			return err
		}
	}

	hooks := &runtime.Hook{}
	hooks.GotLine = term.WriteLine(os.Stdout)
	//if tty {
	//	hooks.GotLine = term.WriteLinePretty(os.Stdout)
	//}

	//var fs runtime.FileSystem
	//if *b != "" {
	fs, err := chroot.New("")
	if err != nil {
		logger.Warn().Err(err).Msg("failed to create runtime filesystem")
	}
	//}

	runtime := runtime.New(
		runtime.WithFileSystem(fs),
		runtime.WithEngine(engine),
		runtime.WithConfig(config),
		runtime.WithHooks(hooks),
	)

	ctx, cancel := context.WithTimeout(ctxmeta, timeout)
	ctx = signal.WithContext(ctx)
	defer cancel()

	err = runtime.Run(ctx)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to run")
	}

	return err
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
