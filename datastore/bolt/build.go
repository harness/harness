package bolt

import (
	"bytes"
	"encoding/binary"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	"github.com/drone/drone/common"
)

// GetBuild gets the specified build number for the
// named repository and build number
func (db *DB) GetBuild(repo string, build int) (*common.Build, error) {
	build_ := &common.Build{}
	key := []byte(repo + "/" + strconv.Itoa(build))

	err := db.View(func(t *bolt.Tx) error {
		return get(t, bucketBuild, key, build_)
	})

	return build_, err
}

// GetBuildList gets a list of recent builds for the
// named repository.
func (db *DB) GetBuildList(repo string) ([]*common.Build, error) {
	// get the last build sequence number (stored in key in `bucketBuildSeq`)
	// get all builds where build number > sequent-20
	// github.com/foo/bar/{number}
	return nil, nil
}

// GetBuildLast gets the last executed build for the
// named repository.
func (db *DB) GetBuildLast(repo string) (*common.Build, error) {
	key := []byte(repo)
	build := &common.Build{}
	err := db.View(func(t *bolt.Tx) error {
		raw := t.Bucket(bucketBuildSeq).Get(key)
		num := binary.LittleEndian.Uint32(raw)
		key = []byte(repo + "/" + strconv.FormatUint(uint64(num), 10))
		return get(t, bucketBuild, key, build)
	})
	return build, err
}

// GetBuildStatus gets the named build status for the
// named repository and build number.
func (db *DB) GetBuildStatus(repo string, build int, status string) (*common.Status, error) {
	status_ := &common.Status{}
	key := []byte(repo + "/" + strconv.Itoa(build) + "/" + status)

	err := db.Update(func(t *bolt.Tx) error {
		return update(t, bucketBuildStatus, key, status)
	})

	return status_, err
}

// GetBuildStatusList gets a list of all build statues for
// the named repository and build number.
func (db *DB) GetBuildStatusList(repo string, build int) ([]*common.Status, error) {
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

// InsertBuild inserts a new build for the named repository
func (db *DB) InsertBuild(repo string, build *common.Build) error {
	key := []byte(repo)

	return db.Update(func(t *bolt.Tx) error {
		raw, err := raw(t, bucketBuildSeq, key)

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
		err = t.Bucket(bucketBuildSeq).Put(key, raw)
		if err != nil {
			return err
		}

		// fill out the build structure
		build.Number = int(next_seq)
		build.Created = time.Now().UTC().Unix()

		key = []byte(repo + "/" + strconv.Itoa(build.Number))
		return insert(t, bucketBuild, key, build)
	})
}

// InsertBuildStatus inserts a new build status for the
// named repository and build number. If the status already
// exists an error is returned.
func (db *DB) InsertBuildStatus(repo string, build int, status *common.Status) error {
	key := []byte(repo + "/" + strconv.Itoa(build) + "/" + status.Context)

	return db.Update(func(t *bolt.Tx) error {
		return update(t, bucketBuildStatus, key, status)
	})
}

// UpdateBuild updates an existing build for the named
// repository. If the build already exists and error is
// returned.
func (db *DB) UpdateBuild(repo string, build *common.Build) error {
	key := []byte(repo + "/" + strconv.Itoa(build.Number))
	build.Updated = time.Now().UTC().Unix()

	return db.Update(func(t *bolt.Tx) error {
		return update(t, bucketBuild, key, build)
	})
}
