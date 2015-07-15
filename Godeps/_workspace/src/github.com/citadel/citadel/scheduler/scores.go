package scheduler

import (
	"sort"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/citadel/citadel"
)

type score struct {
	r     *citadel.EngineSnapshot
	score float64
}

type scores []*score

func sortScores(s []*score) {
	sort.Sort(scores(s))
}

func (s scores) Len() int {
	return len(s)
}

func (s scores) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s scores) Less(i, j int) bool {
	var (
		ip = s[i]
		jp = s[j]
	)

	return ip.score > jp.score
}
