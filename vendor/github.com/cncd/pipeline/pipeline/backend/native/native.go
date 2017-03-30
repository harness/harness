package native

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/cncd/pipeline/pipeline/backend"
)

type procContext struct {
	cmd    *exec.Cmd
	output *io.PipeReader
	stdout *io.PipeWriter
}

type engine struct {
	procs   map[string]procContext
	workDir string
	lock    sync.Mutex
	compl   chan struct{}
}

// New returns a new Engine
func New(workspaceDir string) backend.Engine {
	// strip drive letter with semicolon from path - it may confuse some posix tools, like scp
	if runtime.GOOS == "windows" && len(workspaceDir) > 2 && workspaceDir[1] == ':' {
		workspaceDir = workspaceDir[2:]
	}
	return &engine{
		procs:   make(map[string]procContext, 50),
		workDir: workspaceDir,
	}
}

// NewEnv returns a new Engine using the client connection
// environment variables.
func NewEnv(workspaceDir string) (backend.Engine, error) {
	return New(workspaceDir), nil
}

func (e *engine) Setup(conf *backend.Config) error {

	for _, v := range conf.Volumes {
		log.Print("native.Setup volumes", *v)
		if v.Driver != "local" {
			return fmt.Errorf("Unsupported driver '%s' for volume '%s'", v.Driver, v.Name)
		} else if err := os.MkdirAll(filepath.Join(e.workDir, v.Name), 0777); err != nil {
			return err
		}
	}
	return nil
}

func (e *engine) Exec(proc *backend.Step) error {

	command := ""
	if len(proc.Command) > 0 {
		command = proc.Command[0]
	} else if strings.Index(proc.Image, "plugins/") == 0 {
		// fallback for plugins: try run executable drone-<plugin-name>
		e := strings.Index(proc.Image, ":")
		if e == -1 {
			e = len(proc.Image)
		}
		command = "drone-" + proc.Image[len("plugins/"):e]
		var err error
		if command, err = exec.LookPath(command); err != nil {
			return err
		}
	}

	e.lock.Lock()
	hproc, ok := e.procs[proc.Name]
	e.lock.Unlock()
	if !ok {
		hproc = procContext{}
	} else {
		return fmt.Errorf("Internal error - step '%s' is already in progress", proc.Name)
	}

	cmds := proc.Entrypoint
	cmds = append(cmds, command)

	log.Print("native.Exec cmd:", cmds)
	hproc.cmd = exec.Command(cmds[0], cmds[1:]...)

	// prepare runner environment: merge system env with passed env
	env := os.Environ()
	for k, v := range proc.Environment {
		e := fmt.Sprintf("%s=%s", k, e.fixPath(v, proc))
		env = append(env, e)
		//log.Print("env: ", e)
	}
	hproc.cmd.Env = env
	hproc.cmd.Dir = e.fixPath(proc.WorkingDir, proc)

	if err := os.MkdirAll(hproc.cmd.Dir, 0777); err != nil {
		log.Print(err)
	}

	log.Print("native.Exec volumes: ", proc.Volumes)
	log.Print("native.Exec workingdir: ", e.fixPath(proc.WorkingDir, proc))

	// Create temp script file. This is necessary for windows's cmd.exe
	if ciScript, ok := proc.Environment["CI_SCRIPT"]; ok {
		if ciScriptFile, ok := proc.Environment["CI_SCRIPT_FILE"]; ok {
			//log.Print("native.Exec CI_SCRIPT: ", ciScript)
			f, err := os.Create(filepath.Join(hproc.cmd.Dir, ciScriptFile))
			if err != nil {
				return err
			}

			f.Write([]byte(ciScript))
			f.Close()
		}
	}
	hproc.output, hproc.stdout = io.Pipe()
	hproc.cmd.Stdout = hproc.stdout
	hproc.cmd.Stderr = hproc.stdout

	if err := hproc.cmd.Start(); err != nil {
		hproc.output.Close()
		hproc.stdout.Close()
		return fmt.Errorf("native.Exec: cmd.Start(): %v", err)
	}
	e.lock.Lock()
	e.procs[proc.Name] = hproc
	e.lock.Unlock()
	return nil
}

func (e *engine) Kill(proc *backend.Step) error {
	log.Print("native.Kill: ", proc.Name)

	if hproc, ok := e.procs[proc.Name]; !ok {
		return fmt.Errorf("proc '%s' was not executed", proc.Name)
	} else if hproc.cmd != nil {
		return hproc.cmd.Process.Kill()
	}
	return nil
}

func (e *engine) Wait(proc *backend.Step) (*backend.State, error) {

	log.Print("native.Wait: ", proc.Name)
	e.lock.Lock()
	hproc, ok := e.procs[proc.Name]
	e.lock.Unlock()
	if !ok {
		return nil, fmt.Errorf("Wait: Proc '%s' was not executed", proc.Name)
	}
	state := &backend.State{
		Exited:    true,
		ExitCode:  0,
		OOMKilled: false,
	}
	if hproc.cmd != nil && hproc.cmd.Process != nil {
		if err := hproc.cmd.Wait(); err != nil {
			log.Print("native.Wait: ", err)
		}
		if !hproc.cmd.ProcessState.Exited() {
			// TODO
			log.Printf("native.Wait: Proc '%s' does not exited", proc.Name)
		}
		if !hproc.cmd.ProcessState.Success() {
			state.ExitCode = 1
		}
	}
	hproc.stdout.Close()
	e.lock.Lock()
	delete(e.procs, proc.Name)
	e.lock.Unlock()
	if e.compl != nil {
		var t struct{}
		e.compl <- t
	}
	return state, nil
}

func (e *engine) Tail(proc *backend.Step) (io.ReadCloser, error) {
	e.lock.Lock()
	defer e.lock.Unlock()
	hproc, ok := e.procs[proc.Name]
	if !ok {
		return nil, fmt.Errorf("Proc '%s' was not executed", proc.Name)
	} else if hproc.output == nil {
		return nil, fmt.Errorf("No output from proc '%s'", proc.Name)
	}

	return hproc.output, nil
}

func (e *engine) Destroy(conf *backend.Config) error {
	log.Print("native.Destroy ")

	// Trying to kill all active processes
	// Unfortunatelly, golang havn't cross platform capability to kill all child subproccesses
	// http://stackoverflow.com/questions/22470193/
	// So this implementation will wait completion of last executed command
	e.lock.Lock()
	procs := make([]procContext, 0, len(e.procs))
	for _, p := range e.procs {
		procs = append(procs, p)
	}
	e.lock.Unlock()
	for _, p := range procs {
		if p.cmd != nil && p.cmd.Process != nil {
			p.cmd.Process.Kill()
		}
	}

	vols := make([]string, 0, len(conf.Volumes))
	for _, v := range conf.Volumes {
		if v.Driver == "local" {
			if v.Name == `` || v.Name == `/` {
				log.Printf("Cowardly refuse remove volume with empty name")
			} else {
				vols = append(vols, filepath.Join(e.workDir, v.Name))
			}
		} else {
			log.Printf("Unsupported driver '%s' for volume '%s'", v.Driver, v.Name)
		}
	}

	e.compl = make(chan struct{})
	// Cleanup
	go func() {
		// Wait for completion of all processes
		for {
			e.lock.Lock()
			l := len(e.procs)
			e.lock.Unlock()
			if l != 0 {
				<-e.compl
			} else {
				break
			}
		}
		// Remove volume's dir
		for _, v := range vols {
			log.Print("native.Destroy: removing: ", v)
			os.RemoveAll(v)
		}
		e.compl = nil
	}()

	return nil
}

// fixPath convert path from in-docker path to path on host
func (e *engine) fixPath(p string, proc *backend.Step) string {
	for _, vm := range proc.Volumes {
		if pmap := strings.SplitN(vm, ":", 2); len(pmap) > 1 && len(pmap[1]) > 0 {
			if strings.Index(p, pmap[1]) == 0 {
				p = strings.Replace(p, pmap[1], pmap[0], 1)
				p = strings.Replace(p, ":", "-", -1)
				p = path.Join(e.workDir, p)
				break
			}
		}
	}
	return p
}
