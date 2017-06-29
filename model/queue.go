package model

import (
	"context"

	"github.com/Sirupsen/logrus"
	"github.com/cncd/queue"
)

// Task defines scheduled pipeline Task.
type Task struct {
	ID     string            `meddler:"task_id"`
	Data   []byte            `meddler:"task_data"`
	Labels map[string]string `meddler:"task_labels,json"`
}

// TaskStore defines storage for scheduled Tasks.
type TaskStore interface {
	TaskList() ([]*Task, error)
	TaskInsert(*Task) error
	TaskDelete(string) error
}

// WithTaskStore returns a queue that is backed by the TaskStore. This
// ensures the task Queue can be restored when the system starts.
func WithTaskStore(q queue.Queue, s TaskStore) queue.Queue {
	tasks, _ := s.TaskList()
	for _, task := range tasks {
		q.Push(context.Background(), &queue.Task{
			ID:     task.ID,
			Data:   task.Data,
			Labels: task.Labels,
		})
	}
	return &persistentQueue{q, s}
}

type persistentQueue struct {
	queue.Queue
	store TaskStore
}

// Push pushes an task to the tail of this queue.
func (q *persistentQueue) Push(c context.Context, task *queue.Task) error {
	q.store.TaskInsert(&Task{
		ID:     task.ID,
		Data:   task.Data,
		Labels: task.Labels,
	})
	err := q.Queue.Push(c, task)
	if err != nil {
		q.store.TaskDelete(task.ID)
	}
	return err
}

// Poll retrieves and removes a task head of this queue.
func (q *persistentQueue) Poll(c context.Context, f queue.Filter) (*queue.Task, error) {
	task, err := q.Queue.Poll(c, f)
	if task != nil {
		logrus.Debugf("pull queue item: %s: remove from backup", task.ID)
		if derr := q.store.TaskDelete(task.ID); derr != nil {
			logrus.Errorf("pull queue item: %s: failed to remove from backup: %s", task.ID, derr)
		} else {
			logrus.Debugf("pull queue item: %s: successfully removed from backup", task.ID)
		}
	}
	return task, err
}

// Evict removes a pending task from the queue.
func (q *persistentQueue) Evict(c context.Context, id string) error {
	err := q.Queue.Evict(c, id)
	if err == nil {
		q.store.TaskDelete(id)
	}
	return err
}
