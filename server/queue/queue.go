package queue

import (
	"github.com/drone/drone/server/database"
	"github.com/drone/drone/shared/model"

	"github.com/drone/drone/shared/build/script"
)

// A Queue dispatches tasks to workers.
type Queue struct {
	tasks chan<- *BuildTask
}

// BuildTasks represents a build that is pending
// execution.
type BuildTask struct {
	User   *model.User
	Repo   *model.Repo
	Commit *model.Commit

	// Build instructions from the .drone.yml
	// file, unmarshalled.
	Script *script.Build
}

// Start N workers with the given build runner.
func Start(workers int, commits database.CommitManager, runner BuildRunner) *Queue {
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
