package model

// ProcStore persists process information to storage.
type ProcStore interface {
	ProcLoad(int64) (*Proc, error)
	ProcFind(*Build, int) (*Proc, error)
	ProcChild(*Build, int, string) (*Proc, error)
	ProcList(*Build) ([]*Proc, error)
	ProcCreate([]*Proc) error
	ProcUpdate(*Proc) error
	ProcClear(*Build) error
}

// Proc represents a process in the build pipeline.
// swagger:model proc
type Proc struct {
	ID       int64             `json:"id"                   meddler:"proc_id,pk"`
	BuildID  int64             `json:"build_id"             meddler:"proc_build_id"`
	PID      int               `json:"pid"                  meddler:"proc_pid"`
	PPID     int               `json:"ppid"                 meddler:"proc_ppid"`
	PGID     int               `json:"pgid"                 meddler:"proc_pgid"`
	Name     string            `json:"name"                 meddler:"proc_name"`
	State    string            `json:"state"                meddler:"proc_state"`
	Error    string            `json:"error,omitempty"      meddler:"proc_error"`
	ExitCode int               `json:"exit_code"            meddler:"proc_exit_code"`
	Started  int64             `json:"start_time,omitempty" meddler:"proc_started"`
	Stopped  int64             `json:"end_time,omitempty"   meddler:"proc_stopped"`
	Machine  string            `json:"machine,omitempty"    meddler:"proc_machine"`
	Platform string            `json:"platform,omitempty"   meddler:"proc_platform"`
	Environ  map[string]string `json:"environ,omitempty"    meddler:"proc_environ,json"`
	Children []*Proc           `json:"children,omitempty"   meddler:"-"`
}

// Running returns true if the process state is pending or running.
func (p *Proc) Running() bool {
	return p.State == StatusPending || p.State == StatusRunning
}

// Failing returns true if the process state is failed, killed or error.
func (p *Proc) Failing() bool {
	return p.State == StatusError || p.State == StatusKilled || p.State == StatusFailure
}

// Tree creates a process tree from a flat process list.
func Tree(procs []*Proc) []*Proc {
	var (
		nodes  []*Proc
		parent *Proc
	)
	for _, proc := range procs {
		if proc.PPID == 0 {
			nodes = append(nodes, proc)
			parent = proc
			continue
		} else {
			parent.Children = append(parent.Children, proc)
		}
	}
	return nodes
}
