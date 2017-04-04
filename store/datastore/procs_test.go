package datastore

import (
	"testing"

	"github.com/drone/drone/model"
)

func TestProcFind(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from procs")
		s.Close()
	}()

	err := s.ProcCreate([]*model.Proc{
		{
			BuildID:  1000,
			PID:      1,
			PPID:     2,
			PGID:     3,
			Name:     "build",
			State:    model.StatusSuccess,
			Error:    "pc load letter",
			ExitCode: 255,
			Machine:  "localhost",
			Platform: "linux/amd64",
			Environ:  map[string]string{"GOLANG": "tip"},
		},
	})
	if err != nil {
		t.Errorf("Unexpected error: insert procs: %s", err)
		return
	}

	proc, err := s.ProcFind(&model.Build{ID: 1000}, 1)
	if err != nil {
		t.Error(err)
		return
	}
	if got, want := proc.BuildID, int64(1000); got != want {
		t.Errorf("Want proc fk %d, got %d", want, got)
	}
	if got, want := proc.ID, int64(1); got != want {
		t.Errorf("Want proc pk %d, got %d", want, got)
	}
	if got, want := proc.PID, 1; got != want {
		t.Errorf("Want proc ppid %d, got %d", want, got)
	}
	if got, want := proc.PPID, 2; got != want {
		t.Errorf("Want proc ppid %d, got %d", want, got)
	}
	if got, want := proc.PGID, 3; got != want {
		t.Errorf("Want proc pgid %d, got %d", want, got)
	}
	if got, want := proc.Name, "build"; got != want {
		t.Errorf("Want proc name %s, got %s", want, got)
	}
}

func TestProcChild(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from procs")
		s.Close()
	}()

	err := s.ProcCreate([]*model.Proc{
		{
			BuildID: 1,
			PID:     1,
			PPID:    1,
			PGID:    1,
			State:   "success",
		},
		{
			BuildID: 1,
			PID:     2,
			PGID:    2,
			PPID:    1,
			Name:    "build",
			State:   "success",
		},
	})
	if err != nil {
		t.Errorf("Unexpected error: insert procs: %s", err)
		return
	}
	proc, err := s.ProcChild(&model.Build{ID: 1}, 1, "build")
	if err != nil {
		t.Error(err)
		return
	}

	if got, want := proc.PID, 2; got != want {
		t.Errorf("Want proc pid %d, got %d", want, got)
	}
	if got, want := proc.Name, "build"; got != want {
		t.Errorf("Want proc name %s, got %s", want, got)
	}
}

func TestProcList(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from procs")
		s.Close()
	}()

	err := s.ProcCreate([]*model.Proc{
		{
			BuildID: 2,
			PID:     1,
			PPID:    1,
			PGID:    1,
			State:   "success",
		},
		{
			BuildID: 1,
			PID:     1,
			PPID:    1,
			PGID:    1,
			State:   "success",
		},
		{
			BuildID: 1,
			PID:     2,
			PGID:    2,
			PPID:    1,
			Name:    "build",
			State:   "success",
		},
	})
	if err != nil {
		t.Errorf("Unexpected error: insert procs: %s", err)
		return
	}
	procs, err := s.ProcList(&model.Build{ID: 1})
	if err != nil {
		t.Error(err)
		return
	}
	if got, want := len(procs), 2; got != want {
		t.Errorf("Want %d procs, got %d", want, got)
	}
}

func TestProcUpdate(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from procs")
		s.Close()
	}()

	proc := &model.Proc{
		BuildID:  1,
		PID:      1,
		PPID:     2,
		PGID:     3,
		Name:     "build",
		State:    "pending",
		Error:    "pc load letter",
		ExitCode: 255,
		Machine:  "localhost",
		Platform: "linux/amd64",
		Environ:  map[string]string{"GOLANG": "tip"},
	}
	if err := s.ProcCreate([]*model.Proc{proc}); err != nil {
		t.Errorf("Unexpected error: insert proc: %s", err)
		return
	}
	proc.State = "running"
	if err := s.ProcUpdate(proc); err != nil {
		t.Errorf("Unexpected error: update proc: %s", err)
		return
	}
	updated, err := s.ProcFind(&model.Build{ID: 1}, 1)
	if err != nil {
		t.Error(err)
		return
	}
	if got, want := updated.State, "running"; got != want {
		t.Errorf("Want proc name %s, got %s", want, got)
	}
}

func TestProcIndexes(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from procs")
		s.Close()
	}()

	if err := s.ProcCreate([]*model.Proc{
		{
			BuildID: 1,
			PID:     1,
			PPID:    1,
			PGID:    1,
			State:   "running",
			Name:    "build",
		},
	}); err != nil {
		t.Errorf("Unexpected error: insert procs: %s", err)
		return
	}

	// fail due to duplicate pid
	if err := s.ProcCreate([]*model.Proc{
		{
			BuildID: 1,
			PID:     1,
			PPID:    1,
			PGID:    1,
			State:   "success",
			Name:    "clone",
		},
	}); err == nil {
		t.Errorf("Unexpected error: dupliate pid")
	}

	// // fail due to duplicate process name
	// if err := s.ProcCreate([]*model.Proc{
	// 	{
	// 		BuildID: 1,
	// 		PID:     2,
	// 		PPID:    1,
	// 		PGID:    1,
	// 		State:   "success",
	// 		Name:    "build",
	// 	},
	// }); err == nil {
	// 	t.Errorf("Unexpected error: dupliate name")
	// }
}

// func TestProcCascade(t *testing.T) {
//
// }
