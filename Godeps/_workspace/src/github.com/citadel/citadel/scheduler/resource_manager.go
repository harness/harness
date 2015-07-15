package scheduler

import (
	"fmt"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/citadel/citadel"
)

// ResourceManager is responsible for managing the engines of the cluster
type ResourceManager struct {
}

func NewResourceManager() *ResourceManager {
	return &ResourceManager{}
}

// PlaceImage uses the provided engines to make a decision on which resource the container
// should run based on best utilization of the engines.
func (r *ResourceManager) PlaceContainer(c *citadel.Container, engines []*citadel.EngineSnapshot) (*citadel.EngineSnapshot, error) {
	scores := []*score{}

	for _, e := range engines {
		if e.Memory < c.Image.Memory || e.Cpus < c.Image.Cpus {
			continue
		}

		var (
			cpuScore    = ((e.ReservedCpus + c.Image.Cpus) / e.Cpus) * 100.0
			memoryScore = ((e.ReservedMemory + c.Image.Memory) / e.Memory) * 100.0
			total       = ((cpuScore + memoryScore) / 200.0) * 100.0
		)

		if total <= 100.0 {
			scores = append(scores, &score{r: e, score: total})
		}
	}

	if len(scores) == 0 {
		return nil, fmt.Errorf("no resources avaliable to schedule container")
	}

	sortScores(scores)

	return scores[0].r, nil
}
