package agent

import (
	"context"
	"io"
	"log"
	"math"
	"net/url"
	"sync"
	"time"

	"github.com/cncd/pipeline/pipeline"
	"github.com/cncd/pipeline/pipeline/backend"
	"github.com/cncd/pipeline/pipeline/backend/docker"
	"github.com/cncd/pipeline/pipeline/interrupt"
	"github.com/cncd/pipeline/pipeline/multipart"
	"github.com/cncd/pipeline/pipeline/rpc"

	"github.com/tevino/abool"
	"github.com/urfave/cli"
)

// AgentCmd is the exported command for starting the drone agent.
var AgentCmd = cli.Command{
	Name:   "agent",
	Usage:  "starts the drone agent",
	Action: loop,
	Flags: []cli.Flag{
		cli.StringFlag{
			EnvVar: "DRONE_SERVER,DRONE_ENDPOINT",
			Name:   "drone-server",
			Usage:  "drone server address",
			Value:  "ws://localhost:8000/ws/broker",
		},
		cli.StringFlag{
			EnvVar: "DRONE_SECRET,DRONE_AGENT_SECRET",
			Name:   "drone-secret",
			Usage:  "drone agent secret",
		},
		cli.DurationFlag{
			EnvVar: "DRONE_BACKOFF",
			Name:   "backoff",
			Usage:  "drone server backoff interval",
			Value:  time.Second * 15,
		},
		cli.IntFlag{
			Name:   "retry-limit",
			EnvVar: "DRONE_RETRY_LIMIT",
			Value:  math.MaxInt32,
		},
		cli.BoolFlag{
			EnvVar: "DRONE_DEBUG",
			Name:   "debug",
			Usage:  "start the agent in debug mode",
		},
		cli.StringFlag{
			EnvVar: "DRONE_FILTER",
			Name:   "filter",
			Usage:  "filter jobs processed by this agent",
		},
		cli.IntFlag{
			Name:   "max-procs",
			EnvVar: "DRONE_MAX_PROCS",
			Value:  1,
		},
		cli.StringFlag{
			Name:   "platform",
			EnvVar: "DRONE_PLATFORM",
			Value:  "linux/amd64",
		},
	},
}

func loop(c *cli.Context) error {
	endpoint, err := url.Parse(
		c.String("drone-server"),
	)
	if err != nil {
		return err
	}
	filter := rpc.Filter{
		Labels: map[string]string{
			"platform": c.String("platform"),
		},
	}

	client, err := rpc.NewClient(
		endpoint.String(),
		rpc.WithRetryLimit(
			c.Int("retry-limit"),
		),
		rpc.WithBackoff(
			c.Duration("backoff"),
		),
		rpc.WithToken(
			c.String("drone-secret"),
		),
	)
	if err != nil {
		return err
	}
	defer client.Close()

	sigterm := abool.New()
	ctx := context.Background()
	ctx = interrupt.WithContextFunc(ctx, func() {
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
				if err := run(ctx, client, filter); err != nil {
					log.Printf("build runner encountered error: exiting: %s", err)
					return
				}
			}
		}()
	}

	wg.Wait()
	return nil
}

const (
	maxFileUpload = 5000000
	maxLogsUpload = 5000000
)

func run(ctx context.Context, client rpc.Peer, filter rpc.Filter) error {
	log.Println("pipeline: request next execution")

	// get the next job from the queue
	work, err := client.Next(ctx, filter)
	if err != nil {
		return err
	}
	if work == nil {
		return nil
	}
	log.Printf("pipeline: received next execution: %s", work.ID)

	// new docker engine
	engine, err := docker.NewEnv()
	if err != nil {
		return err
	}

	timeout := time.Hour
	if minutes := work.Timeout; minutes != 0 {
		timeout = time.Duration(minutes) * time.Minute
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cancelled := abool.New()
	go func() {
		if werr := client.Wait(ctx, work.ID); werr != nil {
			cancelled.SetTo(true)
			log.Printf("pipeline: cancel signal received: %s: %s", work.ID, werr)
			cancel()
		} else {
			log.Printf("pipeline: cancel channel closed: %s", work.ID)
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Printf("pipeline: cancel ping loop: %s", work.ID)
				return
			case <-time.After(time.Minute):
				log.Printf("pipeline: ping queue: %s", work.ID)
				client.Extend(ctx, work.ID)
			}
		}
	}()

	state := rpc.State{}
	state.Started = time.Now().Unix()
	err = client.Update(context.Background(), work.ID, state)
	if err != nil {
		log.Printf("pipeline: error updating pipeline status: %s: %s", work.ID, err)
	}

	var uploads sync.WaitGroup
	defaultLogger := pipeline.LogFunc(func(proc *backend.Step, rc multipart.Reader) error {
		part, rerr := rc.NextPart()
		if rerr != nil {
			return rerr
		}
		uploads.Add(1)
		writer := rpc.NewLineWriter(client, work.ID, proc.Alias)
		rlimit := io.LimitReader(part, maxLogsUpload)
		io.Copy(writer, rlimit)

		defer func() {
			log.Printf("pipeline: finish uploading logs: %s: step %s", work.ID, proc.Alias)
			uploads.Done()
		}()

		part, rerr = rc.NextPart()
		if rerr != nil {
			return nil
		}
		rlimit = io.LimitReader(part, maxFileUpload)
		mime := part.Header().Get("Content-Type")
		if serr := client.Upload(context.Background(), work.ID, mime, rlimit); serr != nil {
			log.Printf("pipeline: cannot upload artifact: %s: %s: %s", work.ID, mime, serr)
		}
		return nil
	})

	err = pipeline.New(work.Config,
		pipeline.WithContext(ctx),
		pipeline.WithLogger(defaultLogger),
		pipeline.WithTracer(pipeline.DefaultTracer),
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

	log.Printf("pipeline: execution complete: %s", work.ID)

	uploads.Wait()
	err = client.Update(context.Background(), work.ID, state)
	if err != nil {
		log.Printf("Pipeine: error updating pipeline status: %s: %s", work.ID, err)
	}

	return nil
}
