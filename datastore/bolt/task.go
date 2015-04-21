package bolt

import (
	"bytes"
	"github.com/boltdb/bolt"
	//"github.com/drone/drone/common"
	"io"
	"strconv"
)

/*
	Brad Rydzewski1:00 PM
	the `Task`, `TaskList` and `SetTask` are deprecated and can be probably be removed.
	I just need to make sure we aren't still using those functions anywhere else in the code
*/
/*
// GetTask gets the task at index N for the named
// repository and build number.
func (db *DB) Task(repo string, build int, task int) (*common.Task, error) {
	key := []byte(repo + "/" + strconv.Itoa(build) + "/" + strconv.Itoa(task))
	task_ := &common.Task{}
	err := db.View(func(t *bolt.Tx) error {
		return get(t, bucketBuildTasks, key, task_)
	})
	return task_, err
}

// TaskList gets all tasks for the named repository
// and build number.
func (db *DB) TaskList(repo string, build int) ([]*common.Task, error) {
	// fetch the build so that we can get the
	// number of child tasks.
	build_, err := db.Build(repo, build)
	if err != nil {
		return nil, err
	}

	t, err := db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer t.Rollback()

	// based on the number of child tasks, incrment
	// and loop to get each task from the bucket.
	tasks := []*common.Task{}
	for i := 1; i <= build_.Number; i++ {
		key := []byte(repo + "/" + strconv.Itoa(build) + "/" + strconv.Itoa(i))
		raw := t.Bucket(bucketBuildTasks).Get(key)
		if raw == nil {
			return nil, ErrKeyNotFound
		}
		task := &common.Task{}
		err := decode(raw, task)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// SetTask inserts or updates a task for the named
// repository and build number.
func (db *DB) SetTask(repo string, build int, task *common.Task) error {
	key := []byte(repo + "/" + strconv.Itoa(build) + "/" + strconv.Itoa(task.Number))
	return db.Update(func(t *bolt.Tx) error {
		return update(t, bucketBuildTasks, key, task)
	})
}
*/

// SetLogs inserts or updates a task logs for the
// named repository and build number.
func (db *DB) SetLogs(repo string, build int, task int, log []byte) error {
	key := []byte(repo + "/" + strconv.Itoa(build) + "/" + strconv.Itoa(task))
	t, err := db.Begin(true)
	if err != nil {
		return err
	}
	err = t.Bucket(bucketBuildLogs).Put(key, log)
	if err != nil {
		t.Rollback()
		return err
	}
	return t.Commit()
}

// LogReader gets the task logs at index N for
// the named repository and build number.
func (db *DB) LogReader(repo string, build int, task int) (io.Reader, error) {
	key := []byte(repo + "/" + strconv.Itoa(build) + "/" + strconv.Itoa(task))

	var log []byte
	err := db.View(func(t *bolt.Tx) error {
		var err error
		log, err = raw(t, bucketBuildLogs, key)
		return err
	})
	buf := bytes.NewBuffer(log)
	return buf, err
}
