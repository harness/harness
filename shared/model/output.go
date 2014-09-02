package model

type Output struct {
	ID       int64  `meddler:"output_id,pk"`
	CommitID int64  `meddler:"commit_id"`
	Raw      []byte `meddler:"output_raw"`
}
