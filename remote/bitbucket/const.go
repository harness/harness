package bitbucket

import "github.com/drone/drone/model"

const (
	statusPending = "INPROGRESS"
	statusSuccess = "SUCCESSFUL"
	statusFailure = "FAILED"
)

const (
	descPending = "this build is pending"
	descSuccess = "the build was successful"
	descFailure = "the build failed"
	descError   = "oops, something went wrong"
)

func getStatus(status string) string {
	switch status {
	case model.StatusPending, model.StatusRunning:
		return statusPending
	case model.StatusSuccess:
		return statusSuccess
	default:
		return statusFailure
	}
}

func getDesc(status string) string {
	switch status {
	case model.StatusPending, model.StatusRunning:
		return descPending
	case model.StatusSuccess:
		return descSuccess
	case model.StatusFailure:
		return descFailure
	default:
		return descError
	}
}
