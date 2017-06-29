package main

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"

	"github.com/cncd/pipeline/pipeline"
	"github.com/cncd/pipeline/pipeline/backend"
	"github.com/cncd/pipeline/pipeline/backend/docker"
	"github.com/cncd/pipeline/pipeline/interrupt"
	"github.com/cncd/pipeline/pipeline/multipart"
	"github.com/cncd/pipeline/pipeline/rpc"

	"github.com/tevino/abool"
	"github.com/urfave/cli"
	oldcontext "golang.org/x/net/context"
)

func loop(c *cli.Context) error {
	filter := rpc.Filter{
		Labels: map[string]string{
			"platform": c.String("platform"),
		},
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
	err = client.Init(context.Background(), work.ID, state)
	if err != nil {
		log.Printf("pipeline: error signaling pipeline init: %s: %s", work.ID, err)
	}

	var uploads sync.WaitGroup
	defaultLogger := pipeline.LogFunc(func(proc *backend.Step, rc multipart.Reader) error {
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

		limitedPart := io.LimitReader(part, maxLogsUpload)
		logstream := rpc.NewLineWriter(client, work.ID, proc.Alias, secrets...)
		io.Copy(logstream, limitedPart)

		file := &rpc.File{}
		file.Mime = "application/json+logs"
		file.Proc = proc.Alias
		file.Name = "logs.json"
		file.Data, _ = json.Marshal(logstream.Lines())
		file.Size = len(file.Data)
		file.Time = time.Now().Unix()

		if serr := client.Upload(context.Background(), work.ID, file); serr != nil {
			log.Printf("pipeline: cannot upload logs: %s: %s: %s", work.ID, file.Mime, serr)
		} else {
			log.Printf("pipeline: finish uploading logs: %s: step %s: %s", file.Mime, work.ID, proc.Alias)
		}

		defer func() {
			log.Printf("pipeline: finish uploading logs: %s: step %s", work.ID, proc.Alias)
			uploads.Done()
		}()

		part, rerr = rc.NextPart()
		if rerr != nil {
			return nil
		}
		// TODO should be configurable
		limitedPart = io.LimitReader(part, maxFileUpload)
		file = &rpc.File{}
		file.Mime = part.Header().Get("Content-Type")
		file.Proc = proc.Alias
		file.Name = part.FileName()
		file.Data, _ = ioutil.ReadAll(limitedPart)
		file.Size = len(file.Data)
		file.Time = time.Now().Unix()

		if serr := client.Upload(context.Background(), work.ID, file); serr != nil {
			log.Printf("pipeline: cannot upload artifact: %s: %s: %s", work.ID, file.Mime, serr)
		} else {
			log.Printf("pipeline: finish uploading artifact: %s: step %s: %s", file.Mime, work.ID, proc.Alias)
		}
		return nil
	})

	defaultTracer := pipeline.TraceFunc(func(state *pipeline.State) error {
		procState := rpc.State{
			Proc:     state.Pipeline.Step.Alias,
			Exited:   state.Process.Exited,
			ExitCode: state.Process.ExitCode,
			Started:  time.Now().Unix(), // TODO do not do this
			Finished: time.Now().Unix(),
		}
		defer func() {
			if uerr := client.Update(context.Background(), work.ID, procState); uerr != nil {
				log.Printf("Pipeine: error updating pipeline step status: %s: %s: %s", work.ID, procState.Proc, uerr)
			}
		}()
		if state.Process.Exited {
			return nil
		}
		if state.Pipeline.Step.Environment == nil {
			state.Pipeline.Step.Environment = map[string]string{}
		}
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

	log.Printf("pipeline: execution complete: %s", work.ID)

	uploads.Wait()

	log.Printf("pipeline: logging complete: %s", work.ID)

	err = client.Done(context.Background(), work.ID, state)
	if err != nil {
		log.Printf("Pipeine: error signaling pipeline done: %s: %s", work.ID, err)
	} else {
		log.Printf("pipeline: done: %s", work.ID)
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
