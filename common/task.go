package common

type Task struct {
	Number   int    `json:"number"`
	State    string `json:"state"`
	ExitCode int    `json:"exit_code"`
	Duration int64  `json:"duration"`
	Started  int64  `json:"started_at"`
	Finished int64  `json:"finished_at"`

	// Environment represents the build environment
	// combination from the matrix.
	Environment map[string]string `json:"environment,omitempty"`
}
