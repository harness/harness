package engine

import "os"

var (
	// name of the build agent container.
	DefaultAgent = "drone/drone-exec:latest"

	// default name of the build agent executable
	DefaultEntrypoint = []string{"/bin/drone-exec"}

	// default argument to invoke build steps
	DefaultBuildArgs = []string{"--pull", "--cache", "--clone", "--build", "--deploy"}

	// default argument to invoke build steps
	DefaultPullRequestArgs = []string{"--pull", "--cache", "--clone", "--build"}

	// default arguments to invoke notify steps
	DefaultNotifyArgs = []string{"--pull", "--notify"}
)

func agentImage() string {
	if os.Getenv("DRONE_EXEC_IMAGE") != "" {
		return os.Getenv("DRONE_EXEC_IMAGE")
	}

	return DefaultAgent
}
