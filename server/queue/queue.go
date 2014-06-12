package queue

import (
	"github.com/drone/drone/server/resource/commit"
	"github.com/drone/drone/server/resource/repo"
	"github.com/drone/drone/server/resource/user"

	"github.com/drone/drone/shared/build/script"
)

// A Queue dispatches tasks to workers.
type Queue struct {
	tasks chan<- *BuildTask
}

// BuildTasks represents a build that is pending
// execution.
type BuildTask struct {
	User   *user.User
	Repo   *repo.Repo
	Commit *commit.Commit

	// Build instructions from the .drone.yml
	// file, unmarshalled.
	Script *script.Build
}

// Start N workers with the given build runner.
func Start(workers int, commits commit.CommitManager, runner BuildRunner) *Queue {
	tasks := make(chan *BuildTask)
	queue := &Queue{tasks: tasks}

	for i := 0; i < workers; i++ {
		worker := worker{
			runner:  runner,
			commits: commits,
		}

		go worker.work(tasks)
	}

	return queue
}

// Add adds the task to the build queue.
func (q *Queue) Add(task *BuildTask) {
	q.tasks <- task
}
