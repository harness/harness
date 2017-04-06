package model

// Limiter defines an interface for limiting repository creation.
// This could be used, for example, to limit repository creation to
// a specific organization or a specific set of users.
type Limiter interface {
	LimitUser(*User) error
	LimitRepo(*User, *Repo) error
	LimitBuild(*User, *Repo, *Build) error
}

// NoLimit impliments the Limiter interface without enforcing any
// actual limits. All limiting functions are no-ops.
type NoLimit struct{}

// LimitUser is a no-op for limiting user creation.
func (NoLimit) LimitUser(*User) error { return nil }

// LimitRepo is a no-op for limiting repo creation.
func (NoLimit) LimitRepo(*User, *Repo) error { return nil }

// LimitBuild is a no-op for limiting build creation.
func (NoLimit) LimitBuild(*User, *Repo, *Build) error { return nil }
