package runner

import (
	"bufio"
	"time"
	"fmt"

	"github.com/drone/drone/engine/runner/parse"

	"golang.org/x/net/context"
)

// NoContext is the default context you should supply if not using your own
// context.Context
var NoContext = context.TODO()

// Tracer defines a tracing function that is invoked prior to creating and
// running the container.
type Tracer func(c *Container) error

// Config defines the configuration for creating the Runner.
type Config struct {
	Tracer Tracer
	Engine Engine

	// Buffer defines the size of the buffer for the channel to which the
	// console output is streamed.
	Buffer uint
}

// Runner creates a build Runner using the specific configuration for the given
// Context and Specification.
func (c *Config) Runner(ctx context.Context, spec *Spec) *Runner {

	// TODO(bradyrdzewski) we should make a copy of the configuration parameters
	// instead of a direct reference. This helps avoid any race conditions or
	//unexpected behavior if the Config changes.
	return &Runner{
		ctx:  ctx,
		conf: c,
		spec: spec,
		errc: make(chan error),
		pipe: newPipe(int(c.Buffer) + 1),
	}
}

type Runner struct {
	ctx  context.Context
	conf *Config
	spec *Spec
	pipe *Pipe
	errc chan (error)

	containers []string
	volumes    []string
	networks   []string
}

// Run starts the build runner but does not wait for it to complete. The Wait
// method will return the exit code and release associated resources once the
// running containers exit.
func (r *Runner) Run() error {

	go func() {
		r.setup()
		err := r.exec(r.spec.Nodes.ListNode)
		r.pipe.Close()
		r.cancel()
		r.teardown()
		r.errc <- err
	}()

	go func() {
		<-r.ctx.Done()
		r.cancel()
	}()

	return nil
}

// Wait waits for the runner to exit.
func (r *Runner) Wait() error {
	return <-r.errc
}

// Pipe returns a Pipe that is connected to the console output stream.
func (r *Runner) Pipe() *Pipe {
	return r.pipe
}

func (r *Runner) exec(node parse.Node) error {
	switch v := node.(type) {
	case *parse.ListNode:
		return r.execList(v)
	case *parse.DeferNode:
		return r.execDefer(v)
	case *parse.ErrorNode:
		return r.execError(v)
	case *parse.RecoverNode:
		return r.execRecover(v)
	case *parse.ParallelNode:
		return r.execParallel(v)
	case *parse.RunNode:
		return r.execRun(v)
	}
	return fmt.Errorf("runner: unexepected node %s", node)
}

func (r *Runner) execList(node *parse.ListNode) error {
	for _, n := range node.Body {
		err := r.exec(n)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) execDefer(node *parse.DeferNode) error {
	err1 := r.exec(node.Body)
	err2 := r.exec(node.Defer)
	if err1 != nil {
		return err1
	}
	return err2
}

func (r *Runner) execError(node *parse.ErrorNode) error {
	err := r.exec(node.Body)
	if err != nil {
		r.exec(node.Defer)
	}
	return err
}

func (r *Runner) execRecover(node *parse.RecoverNode) error {
	r.exec(node.Body)
	return nil
}

func (r *Runner) execParallel(node *parse.ParallelNode) error {
	errc := make(chan error)

	for _, n := range node.Body {
		go func(node parse.Node) {
			errc <- r.exec(node)
		}(n)
	}

	var err error
	for i := 0; i < len(node.Body); i++ {
		select {
		case cerr := <-errc:
			if cerr != nil {
				err = cerr
			}
		}
	}

	return err
}

func (r *Runner) execRun(node *parse.RunNode) error {
	container, err := r.spec.lookupContainer(node.Name)
	if err != nil {
		return err
	}
	if r.conf.Tracer != nil {
		err := r.conf.Tracer(container)
		switch {
		case err == ErrSkip:
			return nil
		case err != nil:
			return err
		}
	}
	// TODO(bradrydzewski) there is potential here for a race condition where
	// the context is cancelled just after this line, resulting in the container
	// still being started.
	if r.ctx.Err() != nil {
		return err
	}

	name, err := r.conf.Engine.ContainerStart(container)
	if err != nil {
		return err
	}
	r.containers = append(r.containers, name)

	go func() {
		if node.Silent {
			return
		}
		rc, err := r.conf.Engine.ContainerLogs(name)
		if err != nil {
			return
		}
		defer rc.Close()

		num := 0
		now := time.Now().UTC()
		scanner := bufio.NewScanner(rc)
		for scanner.Scan() {
			r.pipe.lines <- &Line{
				Proc: container.Alias,
				Time: int64(time.Since(now).Seconds()),
				Pos:  num,
				Out:  scanner.Text(),
			}
			num++
		}
	}()

	// exit when running container in detached mode in background
	if node.Detach {
		return nil
	}

	state, err := r.conf.Engine.ContainerWait(name)
	if err != nil {
		return err
	}
	if state.OOMKilled {
		return &OomError{name}
	} else if state.ExitCode != 0 {
		return &ExitError{name, state.ExitCode}
	}
	return nil
}

func (r *Runner) setup() {
	// this is where we will setup network and volumes
}

func (r *Runner) teardown() {
	// TODO(bradrydzewski) this is not yet thread safe.
	for _, container := range r.containers {
		r.conf.Engine.ContainerRemove(container)
	}
}

func (r *Runner) cancel() {
	// TODO(bradrydzewski) this is not yet thread safe.
	for _, container := range r.containers {
		r.conf.Engine.ContainerStop(container)
	}
}
