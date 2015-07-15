package scheduler

import "github.com/drone/drone/Godeps/_workspace/src/github.com/citadel/citadel"

type MultiScheduler struct {
	schedulers []citadel.Scheduler
}

func NewMultiScheduler(s ...citadel.Scheduler) citadel.Scheduler {
	return &MultiScheduler{
		schedulers: s,
	}
}

func (m *MultiScheduler) Schedule(c *citadel.Image, e *citadel.Engine) (bool, error) {
	for _, s := range m.schedulers {
		canrun, err := s.Schedule(c, e)
		if err != nil {
			return false, err
		}

		if !canrun {
			return false, nil
		}
	}

	return true, nil
}
