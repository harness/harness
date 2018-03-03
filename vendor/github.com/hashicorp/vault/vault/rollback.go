package vault

import (
	"context"
	"strings"
	"sync"
	"time"

	log "github.com/mgutz/logxi/v1"

	"github.com/armon/go-metrics"
	"github.com/hashicorp/vault/logical"
)

const (
	// rollbackPeriod is how often we attempt rollbacks for all the backends
	rollbackPeriod = time.Minute
)

// RollbackManager is responsible for performing rollbacks of partial
// secrets within logical backends.
//
// During normal operations, it is possible for logical backends to
// error partially through an operation. These are called "partial secrets":
// they are never sent back to a user, but they do need to be cleaned up.
// This manager handles that by periodically (on a timer) requesting that the
// backends clean up.
//
// The RollbackManager periodically initiates a logical.RollbackOperation
// on every mounted logical backend. It ensures that only one rollback operation
// is in-flight at any given time within a single seal/unseal phase.
type RollbackManager struct {
	logger log.Logger

	// This gives the current mount table of both logical and credential backends,
	// plus a RWMutex that is locked for reading. It is up to the caller to RUnlock
	// it when done with the mount table.
	backends func() []*MountEntry

	router *Router
	period time.Duration

	inflightAll  sync.WaitGroup
	inflight     map[string]*rollbackState
	inflightLock sync.RWMutex

	doneCh       chan struct{}
	shutdown     bool
	shutdownCh   chan struct{}
	shutdownLock sync.Mutex
	quitContext  context.Context
}

// rollbackState is used to track the state of a single rollback attempt
type rollbackState struct {
	lastError error
	sync.WaitGroup
}

// NewRollbackManager is used to create a new rollback manager
func NewRollbackManager(logger log.Logger, backendsFunc func() []*MountEntry, router *Router, ctx context.Context) *RollbackManager {
	r := &RollbackManager{
		logger:      logger,
		backends:    backendsFunc,
		router:      router,
		period:      rollbackPeriod,
		inflight:    make(map[string]*rollbackState),
		doneCh:      make(chan struct{}),
		shutdownCh:  make(chan struct{}),
		quitContext: ctx,
	}
	return r
}

// Start starts the rollback manager
func (m *RollbackManager) Start() {
	go m.run()
}

// Stop stops the running manager. This will wait for any in-flight
// rollbacks to complete.
func (m *RollbackManager) Stop() {
	m.shutdownLock.Lock()
	defer m.shutdownLock.Unlock()
	if !m.shutdown {
		m.shutdown = true
		close(m.shutdownCh)
		<-m.doneCh
	}
	m.inflightAll.Wait()
}

// run is a long running routine to periodically invoke rollback
func (m *RollbackManager) run() {
	m.logger.Info("rollback: starting rollback manager")
	tick := time.NewTicker(m.period)
	defer tick.Stop()
	defer close(m.doneCh)
	for {
		select {
		case <-tick.C:
			m.triggerRollbacks()

		case <-m.shutdownCh:
			m.logger.Info("rollback: stopping rollback manager")
			return
		}
	}
}

// triggerRollbacks is used to trigger the rollbacks across all the backends
func (m *RollbackManager) triggerRollbacks() {

	backends := m.backends()

	for _, e := range backends {
		path := e.Path
		if e.Table == credentialTableType {
			path = credentialRoutePrefix + path
		}

		// When the mount is filtered, the backend will be nil
		backend := m.router.MatchingBackend(path)
		if backend == nil {
			continue
		}

		m.inflightLock.RLock()
		_, ok := m.inflight[path]
		m.inflightLock.RUnlock()
		if !ok {
			m.startRollback(path)
		}
	}
}

// startRollback is used to start an async rollback attempt.
// This must be called with the inflightLock held.
func (m *RollbackManager) startRollback(path string) *rollbackState {
	rs := &rollbackState{}
	rs.Add(1)
	m.inflightAll.Add(1)
	m.inflightLock.Lock()
	m.inflight[path] = rs
	m.inflightLock.Unlock()
	go m.attemptRollback(m.quitContext, path, rs)
	return rs
}

// attemptRollback invokes a RollbackOperation for the given path
func (m *RollbackManager) attemptRollback(ctx context.Context, path string, rs *rollbackState) (err error) {
	defer metrics.MeasureSince([]string{"rollback", "attempt", strings.Replace(path, "/", "-", -1)}, time.Now())
	if m.logger.IsTrace() {
		m.logger.Trace("rollback: attempting rollback", "path", path)
	}

	defer func() {
		rs.lastError = err
		rs.Done()
		m.inflightAll.Done()
		m.inflightLock.Lock()
		delete(m.inflight, path)
		m.inflightLock.Unlock()
	}()

	// Invoke a RollbackOperation
	req := &logical.Request{
		Operation: logical.RollbackOperation,
		Path:      path,
	}
	_, err = m.router.Route(ctx, req)

	// If the error is an unsupported operation, then it doesn't
	// matter, the backend doesn't support it.
	if err == logical.ErrUnsupportedOperation {
		err = nil
	}
	// If we failed due to read-only storage, we can't do anything; ignore
	if err != nil && strings.Contains(err.Error(), logical.ErrReadOnly.Error()) {
		err = nil
	}
	if err != nil {
		m.logger.Error("rollback: error rolling back", "path", path, "error", err)
	}
	return
}

// Rollback is used to trigger an immediate rollback of the path,
// or to join an existing rollback operation if in flight.
func (m *RollbackManager) Rollback(path string) error {
	// Check for an existing attempt and start one if none
	m.inflightLock.RLock()
	rs, ok := m.inflight[path]
	m.inflightLock.RUnlock()
	if !ok {
		rs = m.startRollback(path)
	}

	// Wait for the attempt to finish
	rs.Wait()

	// Return the last error
	return rs.lastError
}

// The methods below are the hooks from core that are called pre/post seal.

// startRollback is used to start the rollback manager after unsealing
func (c *Core) startRollback() error {
	backendsFunc := func() []*MountEntry {
		ret := []*MountEntry{}
		c.mountsLock.RLock()
		defer c.mountsLock.RUnlock()
		// During teardown/setup after a leader change or unseal there could be
		// something racy here so make sure the table isn't nil
		if c.mounts != nil {
			for _, entry := range c.mounts.Entries {
				ret = append(ret, entry)
			}
		}
		c.authLock.RLock()
		defer c.authLock.RUnlock()
		// During teardown/setup after a leader change or unseal there could be
		// something racy here so make sure the table isn't nil
		if c.auth != nil {
			for _, entry := range c.auth.Entries {
				ret = append(ret, entry)
			}
		}
		return ret
	}
	c.rollback = NewRollbackManager(c.logger, backendsFunc, c.router, c.activeContext)
	c.rollback.Start()
	return nil
}

// stopRollback is used to stop running the rollback manager before sealing
func (c *Core) stopRollback() error {
	if c.rollback != nil {
		c.rollback.Stop()
		c.rollback = nil
	}
	return nil
}
