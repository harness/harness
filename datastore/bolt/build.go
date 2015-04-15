package bolt

import (
	"bytes"
	"encoding/binary"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	"github.com/drone/drone/common"
)

// Build gets the specified build number for the
// named repository and build number.
func (db *DB) Build(repo string, build int) (*common.Build, error) {
	build_ := &common.Build{}
	key := []byte(repo + "/" + strconv.Itoa(build))

	err := db.View(func(t *bolt.Tx) error {
		return get(t, bucketBuild, key, build_)
	})

	return build_, err
}

// BuildList gets a list of recent builds for the
// named repository.
func (db *DB) BuildList(repo string) ([]*common.Build, error) {
	// TODO (bradrydzewski) we can do this more efficiently
	var builds []*common.Build
	build, err := db.BuildLast(repo)
	if err == ErrKeyNotFound {
		return builds, nil
	} else if err != nil {
		return nil, err
	}

	err = db.View(func(t *bolt.Tx) error {
		pos := build.Number - 25
		if pos < 1 {
			pos = 1
		}
		for i := pos; i <= build.Number; i++ {
			key := []byte(repo + "/" + strconv.Itoa(i))
			build := &common.Build{}
			err = get(t, bucketBuild, key, build)
			if err != nil {
				return err
			}
			builds = append(builds, build)
		}
		return nil
	})
	return builds, err
}

// BuildLast gets the last executed build for the
// named repository.
func (db *DB) BuildLast(repo string) (*common.Build, error) {
	key := []byte(repo)
	build := &common.Build{}
	err := db.View(func(t *bolt.Tx) error {
		raw := t.Bucket(bucketBuildSeq).Get(key)
		if raw == nil {
			return ErrKeyNotFound
		}
		num := binary.LittleEndian.Uint32(raw)
		key = []byte(repo + "/" + strconv.FormatUint(uint64(num), 10))
		return get(t, bucketBuild, key, build)
	})
	return build, err
}

// SetBuild inserts or updates a build for the named
// repository. The build number is incremented and
// assigned to the provided build.
func (db *DB) SetBuild(repo string, build *common.Build) error {
	repokey := []byte(repo)
	build.Updated = time.Now().UTC().Unix()

	return db.Update(func(t *bolt.Tx) error {

		if build.Number == 0 {
			raw, err := raw(t, bucketBuildSeq, repokey)

			var next_seq uint32
			switch err {
			case ErrKeyNotFound:
				next_seq = 1
			case nil:
				next_seq = 1 + binary.LittleEndian.Uint32(raw)
			default:
				return err
			}

			// covert our seqno to raw value
			raw = make([]byte, 4) // TODO(benschumacher) replace magic number 4 (uint32)
			binary.LittleEndian.PutUint32(raw, next_seq)
			err = t.Bucket(bucketBuildSeq).Put(repokey, raw)
			if err != nil {
				return err
			}

			// fill out the build structure
			build.Number = int(next_seq)
			build.Created = time.Now().UTC().Unix()
		}

		key := []byte(repo + "/" + strconv.Itoa(build.Number))
		return update(t, bucketBuild, key, build)
	})
}

// Status returns the status for the given repository
// and build number.
func (db *DB) Status(repo string, build int, status string) (*common.Status, error) {
	status_ := &common.Status{}
	key := []byte(repo + "/" + strconv.Itoa(build) + "/" + status)

	err := db.Update(func(t *bolt.Tx) error {
		return update(t, bucketBuildStatus, key, status)
	})

	return status_, err
}

// StatusList returned a list of all build statues for
// the given repository and build number.
func (db *DB) StatusList(repo string, build int) ([]*common.Status, error) {
	// TODO (bradrydzewski) explore efficiency of cursor vs index

	statuses := []*common.Status{}
	err := db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bucketBuildStatus).Cursor()
		prefix := []byte(repo + "/" + strconv.Itoa(build) + "/")
		for k, v := c.Seek(prefix); bytes.HasPrefix(k, prefix); k, v = c.Next() {
			status := &common.Status{}
			if err := decode(v, status); err != nil {
				return err
			}
			statuses = append(statuses, status)
		}
		return nil
	})
	return statuses, err
}

// SetStatus inserts a new build status for the
// named repository and build number. If the status already
// exists an error is returned.
func (db *DB) SetStatus(repo string, build int, status *common.Status) error {
	key := []byte(repo + "/" + strconv.Itoa(build) + "/" + status.Context)

	return db.Update(func(t *bolt.Tx) error {
		return update(t, bucketBuildStatus, key, status)
	})
}
