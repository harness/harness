// +build !ent
// +build !prem
// +build !pro
// +build !hsm

package vault

import (
	"context"
	"fmt"
	"sync/atomic"

	proto "github.com/golang/protobuf/proto"
	"github.com/hashicorp/vault/helper/locksutil"
	"github.com/hashicorp/vault/physical"
	log "github.com/mgutz/logxi/v1"
)

// NewSealUnwrapper creates a new seal unwrapper
func NewSealUnwrapper(underlying physical.Backend, logger log.Logger) physical.Backend {
	ret := &sealUnwrapper{
		underlying:   underlying,
		logger:       logger,
		locks:        locksutil.CreateLocks(),
		allowUnwraps: new(uint32),
	}

	if underTxn, ok := underlying.(physical.Transactional); ok {
		return &transactionalSealUnwrapper{
			sealUnwrapper: ret,
			Transactional: underTxn,
		}
	}

	return ret
}

var _ physical.Backend = (*sealUnwrapper)(nil)
var _ physical.Transactional = (*transactionalSealUnwrapper)(nil)

type sealUnwrapper struct {
	underlying   physical.Backend
	logger       log.Logger
	locks        []*locksutil.LockEntry
	allowUnwraps *uint32
}

// transactionalSealUnwrapper is a seal unwrapper that wraps a physical that is transactional
type transactionalSealUnwrapper struct {
	*sealUnwrapper
	physical.Transactional
}

func (d *sealUnwrapper) Put(ctx context.Context, entry *physical.Entry) error {
	if entry == nil {
		return nil
	}

	locksutil.LockForKey(d.locks, entry.Key).Lock()
	defer locksutil.LockForKey(d.locks, entry.Key).Unlock()

	return d.underlying.Put(ctx, entry)
}

func (d *sealUnwrapper) Get(ctx context.Context, key string) (*physical.Entry, error) {
	entry, err := d.underlying.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}

	var performUnwrap bool
	se := &physical.SealWrapEntry{}
	// If the value ends in our canary value, try to decode the bytes.
	eLen := len(entry.Value)
	if eLen > 0 && entry.Value[eLen-1] == 's' {
		if err := proto.Unmarshal(entry.Value[:eLen-1], se); err == nil {
			// We unmarshaled successfully which means we need to store it as a
			// non-proto message
			performUnwrap = true
		}
	}
	if !performUnwrap {
		return entry, nil
	}
	// It's actually encrypted and we can't read it
	if se.Wrapped {
		return nil, fmt.Errorf("cannot decode sealwrapped storage entry %s", entry.Key)
	}
	if atomic.LoadUint32(d.allowUnwraps) != 1 {
		return &physical.Entry{
			Key:   entry.Key,
			Value: se.Ciphertext,
		}, nil
	}

	locksutil.LockForKey(d.locks, key).Lock()
	defer locksutil.LockForKey(d.locks, key).Unlock()

	// At this point we need to re-read and re-check
	entry, err = d.underlying.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}

	performUnwrap = false
	se = &physical.SealWrapEntry{}
	// If the value ends in our canary value, try to decode the bytes.
	eLen = len(entry.Value)
	if eLen > 0 && entry.Value[eLen-1] == 's' {
		// We ignore an error because the canary is not a guarantee; if it
		// doesn't decode, proceed normally
		if err := proto.Unmarshal(entry.Value[:eLen-1], se); err == nil {
			// We unmarshaled successfully which means we need to store it as a
			// non-proto message
			performUnwrap = true
		}
	}
	if !performUnwrap {
		return entry, nil
	}
	if se.Wrapped {
		return nil, fmt.Errorf("cannot decode sealwrapped storage entry %s", entry.Key)
	}

	entry = &physical.Entry{
		Key:   entry.Key,
		Value: se.Ciphertext,
	}

	if atomic.LoadUint32(d.allowUnwraps) != 1 {
		return entry, nil
	}
	return entry, d.underlying.Put(ctx, entry)
}

func (d *sealUnwrapper) Delete(ctx context.Context, key string) error {
	locksutil.LockForKey(d.locks, key).Lock()
	defer locksutil.LockForKey(d.locks, key).Unlock()

	return d.underlying.Delete(ctx, key)
}

func (d *sealUnwrapper) List(ctx context.Context, prefix string) ([]string, error) {
	return d.underlying.List(ctx, prefix)
}

func (d *transactionalSealUnwrapper) Transaction(ctx context.Context, txns []*physical.TxnEntry) error {
	// Collect keys that need to be locked
	var keys []string
	for _, curr := range txns {
		keys = append(keys, curr.Entry.Key)
	}
	// Lock the keys
	for _, l := range locksutil.LocksForKeys(d.locks, keys) {
		l.Lock()
		defer l.Unlock()
	}

	if err := d.Transactional.Transaction(ctx, txns); err != nil {
		return err
	}

	return nil
}

// This should only run during preSeal which ensures that it can't be run
// concurrently and that it will be run only by the active node
func (d *sealUnwrapper) stopUnwraps() {
	atomic.StoreUint32(d.allowUnwraps, 0)
}

func (d *sealUnwrapper) runUnwraps() {
	// Allow key unwraps on key gets. This gets set only when running on the
	// active node to prevent standbys from changing data underneath the
	// primary
	atomic.StoreUint32(d.allowUnwraps, 1)
}
