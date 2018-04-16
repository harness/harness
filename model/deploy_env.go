package model

// DeployEnvStore persists process information to storage.
type DeployEnvStore interface {
	DeployEnvLoad(int64) (*DeployEnv, error)
	DeployEnvFind(*Build, int) (*DeployEnv, error)
	DeployEnvList(*Build) ([]*DeployEnv, error)
	DeployEnvCreate([]*DeployEnv) error
}

// DeployEnv represents a process in the build pipeline.
// swagger:model depoly_env
type DeployEnv struct {
	ID      int64  `json:"id"                   meddler:"deploy_env_id,pk"`
	BuildID int64  `json:"build_id"             meddler:"deploy_env_build_id"`
	Name    string `json:"name"                 meddler:"deploy_env_name"`
}
