package model

// Gated indicates a build is gated and requires approval to proceed. It also
// includes the reason the build was gated.
type Gated struct {
	Gated  bool   `json:"gated"`
	Reason string `json:"reason"`
}

// GateService defines a service for gating builds.
type GateService interface {
	Gated(*User, *Repo, *Build) (*Gated, error)
}

type gateService struct{}

func (s *gateService) Gated(owner *User, repo *Repo, build *Build) (*Gated, error) {
	g := new(Gated)
	if repo.IsPrivate && build.Event == EventPull && build.Sender != owner.Login {
		g.Gated = true
	}
	return g, nil
}
