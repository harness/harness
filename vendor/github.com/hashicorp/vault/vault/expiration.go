package vault

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/armon/go-metrics"
	log "github.com/mgutz/logxi/v1"

	"github.com/hashicorp/errwrap"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/vault/helper/consts"
	"github.com/hashicorp/vault/helper/jsonutil"
	"github.com/hashicorp/vault/helper/locksutil"
	"github.com/hashicorp/vault/logical"
)

const (
	// expirationSubPath is the sub-path used for the expiration manager
	// view. This is nested under the system view.
	expirationSubPath = "expire/"

	// leaseViewPrefix is the prefix used for the ID based lookup of leases.
	leaseViewPrefix = "id/"

	// tokenViewPrefix is the prefix used for the token based lookup of leases.
	tokenViewPrefix = "token/"

	// maxRevokeAttempts limits how many revoke attempts are made
	maxRevokeAttempts = 6

	// revokeRetryBase is a baseline retry time
	revokeRetryBase = 10 * time.Second

	// maxLeaseDuration is the default maximum lease duration
	maxLeaseTTL = 32 * 24 * time.Hour

	// defaultLeaseDuration is the default lease duration used when no lease is specified
	defaultLeaseTTL = maxLeaseTTL

	//maxLeaseThreshold is the maximum lease count before generating log warning
	maxLeaseThreshold = 256000
)

// ExpirationManager is used by the Core to manage leases. Secrets
// can provide a lease, meaning that they can be renewed or revoked.
// If a secret is not renewed in timely manner, it may be expired, and
// the ExpirationManager will handle doing automatic revocation.
type ExpirationManager struct {
	router     *Router
	idView     *BarrierView
	tokenView  *BarrierView
	tokenStore *TokenStore
	logger     log.Logger

	pending     map[string]*time.Timer
	pendingLock sync.RWMutex

	tidyLock int32

	restoreMode        int32
	restoreModeLock    sync.RWMutex
	restoreRequestLock sync.RWMutex
	restoreLocks       []*locksutil.LockEntry
	restoreLoaded      sync.Map
	quitCh             chan struct{}

	coreStateLock     *sync.RWMutex
	quitContext       context.Context
	leaseCheckCounter uint32
}

// NewExpirationManager creates a new ExpirationManager that is backed
// using a given view, and uses the provided router for revocation.
func NewExpirationManager(c *Core, view *BarrierView) *ExpirationManager {
	exp := &ExpirationManager{
		router:     c.router,
		idView:     view.SubView(leaseViewPrefix),
		tokenView:  view.SubView(tokenViewPrefix),
		tokenStore: c.tokenStore,
		logger:     c.logger,
		pending:    make(map[string]*time.Timer),

		// new instances of the expiration manager will go immediately into
		// restore mode
		restoreMode:  1,
		restoreLocks: locksutil.CreateLocks(),
		quitCh:       make(chan struct{}),

		coreStateLock:     &c.stateLock,
		quitContext:       c.activeContext,
		leaseCheckCounter: 0,
	}

	if exp.logger == nil {
		exp.logger = log.New("expiration_manager")
	}

	return exp
}

// setupExpiration is invoked after we've loaded the mount table to
// initialize the expiration manager
func (c *Core) setupExpiration() error {
	c.metricsMutex.Lock()
	defer c.metricsMutex.Unlock()
	// Create a sub-view
	view := c.systemBarrierView.SubView(expirationSubPath)

	// Create the manager
	mgr := NewExpirationManager(c, view)
	c.expiration = mgr

	// Link the token store to this
	c.tokenStore.SetExpirationManager(mgr)

	// Restore the existing state
	c.logger.Info("expiration: restoring leases")
	errorFunc := func() {
		c.logger.Error("expiration: shutting down")
		if err := c.Shutdown(); err != nil {
			c.logger.Error("expiration: error shutting down core: %v", err)
		}
	}
	go c.expiration.Restore(errorFunc)

	return nil
}

// stopExpiration is used to stop the expiration manager before
// sealing the Vault.
func (c *Core) stopExpiration() error {
	if c.expiration != nil {
		if err := c.expiration.Stop(); err != nil {
			return err
		}
		c.metricsMutex.Lock()
		defer c.metricsMutex.Unlock()
		c.expiration = nil
	}
	return nil
}

// lockLease takes out a lock for a given lease ID
func (m *ExpirationManager) lockLease(leaseID string) {
	locksutil.LockForKey(m.restoreLocks, leaseID).Lock()
}

// unlockLease unlocks a given lease ID
func (m *ExpirationManager) unlockLease(leaseID string) {
	locksutil.LockForKey(m.restoreLocks, leaseID).Unlock()
}

// inRestoreMode returns if we are currently in restore mode
func (m *ExpirationManager) inRestoreMode() bool {
	return atomic.LoadInt32(&m.restoreMode) == 1
}

// Tidy cleans up the dangling storage entries for leases. It scans the storage
// view to find all the available leases, checks if the token embedded in it is
// either empty or invalid and in both the cases, it revokes them. It also uses
// a token cache to avoid multiple lookups of the same token ID. It is normally
// not required to use the API that invokes this. This is only intended to
// clean up the corrupt storage due to bugs.
func (m *ExpirationManager) Tidy() error {
	if m.inRestoreMode() {
		return errors.New("cannot run tidy while restoring leases")
	}

	var tidyErrors *multierror.Error

	if !atomic.CompareAndSwapInt32(&m.tidyLock, 0, 1) {
		m.logger.Warn("expiration: tidy operation on leases is already in progress")
		return fmt.Errorf("tidy operation on leases is already in progress")
	}

	defer atomic.CompareAndSwapInt32(&m.tidyLock, 1, 0)

	m.logger.Info("expiration: beginning tidy operation on leases")
	defer m.logger.Info("expiration: finished tidy operation on leases")

	// Create a cache to keep track of looked up tokens
	tokenCache := make(map[string]bool)
	var countLease, revokedCount, deletedCountInvalidToken, deletedCountEmptyToken int64

	tidyFunc := func(leaseID string) {
		countLease++
		if countLease%500 == 0 {
			m.logger.Info("expiration: tidying leases", "progress", countLease)
		}

		le, err := m.loadEntry(leaseID)
		if err != nil {
			tidyErrors = multierror.Append(tidyErrors, fmt.Errorf("failed to load the lease ID %q: %v", leaseID, err))
			return
		}

		if le == nil {
			tidyErrors = multierror.Append(tidyErrors, fmt.Errorf("nil entry for lease ID %q: %v", leaseID, err))
			return
		}

		var isValid, ok bool
		revokeLease := false
		if le.ClientToken == "" {
			m.logger.Trace("expiration: revoking lease which has an empty token", "lease_id", leaseID)
			revokeLease = true
			deletedCountEmptyToken++
			goto REVOKE_CHECK
		}

		isValid, ok = tokenCache[le.ClientToken]
		if !ok {
			saltedID, err := m.tokenStore.SaltID(le.ClientToken)
			if err != nil {
				tidyErrors = multierror.Append(tidyErrors, fmt.Errorf("failed to lookup salt id: %v", err))
				return
			}
			lock := locksutil.LockForKey(m.tokenStore.tokenLocks, le.ClientToken)
			lock.RLock()
			te, err := m.tokenStore.lookupSalted(m.quitContext, saltedID, true)
			lock.RUnlock()

			if err != nil {
				tidyErrors = multierror.Append(tidyErrors, fmt.Errorf("failed to lookup token: %v", err))
				return
			}

			if te == nil {
				m.logger.Trace("expiration: revoking lease which holds an invalid token", "lease_id", leaseID)
				revokeLease = true
				deletedCountInvalidToken++
				tokenCache[le.ClientToken] = false
			} else {
				tokenCache[le.ClientToken] = true
			}
			goto REVOKE_CHECK
		} else {
			if isValid {
				return
			}

			m.logger.Trace("expiration: revoking lease which contains an invalid token", "lease_id", leaseID)
			revokeLease = true
			deletedCountInvalidToken++
			goto REVOKE_CHECK
		}

	REVOKE_CHECK:
		if revokeLease {
			// Force the revocation and skip going through the token store
			// again
			err = m.revokeCommon(leaseID, true, true)
			if err != nil {
				tidyErrors = multierror.Append(tidyErrors, fmt.Errorf("failed to revoke an invalid lease with ID %q: %v", leaseID, err))
				return
			}
			revokedCount++
		}
	}

	if err := logical.ScanView(m.quitContext, m.idView, tidyFunc); err != nil {
		return err
	}

	m.logger.Debug("expiration: number of leases scanned", "count", countLease)
	m.logger.Debug("expiration: number of leases which had empty tokens", "count", deletedCountEmptyToken)
	m.logger.Debug("expiration: number of leases which had invalid tokens", "count", deletedCountInvalidToken)
	m.logger.Debug("expiration: number of leases successfully revoked", "count", revokedCount)

	return tidyErrors.ErrorOrNil()
}

// Restore is used to recover the lease states when starting.
// This is used after starting the vault.
func (m *ExpirationManager) Restore(errorFunc func()) (retErr error) {
	defer func() {
		// Turn off restore mode. We can do this safely without the lock because
		// if restore mode finished successfully, restore mode was already
		// disabled with the lock. In an error state, this will allow the
		// Stop() function to shut everything down.
		atomic.StoreInt32(&m.restoreMode, 0)

		switch {
		case retErr == nil:
		case errwrap.Contains(retErr, ErrBarrierSealed.Error()):
			// Don't run error func because we're likely already shutting down
			m.logger.Warn("expiration: barrier sealed while restoring leases, stopping lease loading")
			retErr = nil
		default:
			m.logger.Error("expiration: error restoring leases", "error", retErr)
			if errorFunc != nil {
				errorFunc()
			}
		}
	}()

	// Accumulate existing leases
	m.logger.Debug("expiration: collecting leases")
	existing, err := logical.CollectKeys(m.quitContext, m.idView)
	if err != nil {
		return errwrap.Wrapf("failed to scan for leases: {{err}}", err)
	}
	m.logger.Debug("expiration: leases collected", "num_existing", len(existing))

	// Make the channels used for the worker pool
	broker := make(chan string)
	quit := make(chan bool)
	// Buffer these channels to prevent deadlocks
	errs := make(chan error, len(existing))
	result := make(chan struct{}, len(existing))

	// Use a wait group
	wg := &sync.WaitGroup{}

	// Create 64 workers to distribute work to
	for i := 0; i < consts.ExpirationRestoreWorkerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case leaseID, ok := <-broker:
					// broker has been closed, we are done
					if !ok {
						return
					}

					err := m.processRestore(leaseID)
					if err != nil {
						errs <- err
						continue
					}

					// Send message that lease is done
					result <- struct{}{}

				// quit early
				case <-quit:
					return

				case <-m.quitCh:
					return
				}
			}
		}()
	}

	// Distribute the collected keys to the workers in a go routine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i, leaseID := range existing {
			if i > 0 && i%500 == 0 {
				m.logger.Trace("expiration: leases loading", "progress", i)
			}

			select {
			case <-quit:
				return

			case <-m.quitCh:
				return

			default:
				broker <- leaseID
			}
		}

		// Close the broker, causing worker routines to exit
		close(broker)
	}()

	// Ensure all keys on the chan are processed
	for i := 0; i < len(existing); i++ {
		select {
		case err := <-errs:
			// Close all go routines
			close(quit)
			return err

		case <-m.quitCh:
			close(quit)
			return nil

		case <-result:
		}
	}

	// Let all go routines finish
	wg.Wait()

	m.restoreModeLock.Lock()
	m.restoreLoaded = sync.Map{}
	m.restoreLocks = nil
	atomic.StoreInt32(&m.restoreMode, 0)
	m.restoreModeLock.Unlock()

	m.logger.Info("expiration: lease restore complete")
	return nil
}

// processRestore takes a lease and restores it in the expiration manager if it has
// not already been seen
func (m *ExpirationManager) processRestore(leaseID string) error {
	m.restoreRequestLock.RLock()
	defer m.restoreRequestLock.RUnlock()

	// Check if the lease has been seen
	if _, ok := m.restoreLoaded.Load(leaseID); ok {
		return nil
	}

	m.lockLease(leaseID)
	defer m.unlockLease(leaseID)

	// Check again with the lease locked
	if _, ok := m.restoreLoaded.Load(leaseID); ok {
		return nil
	}

	// Load lease and restore expiration timer
	_, err := m.loadEntryInternal(leaseID, true, false)
	if err != nil {
		return err
	}
	return nil
}

// Stop is used to prevent further automatic revocations.
// This must be called before sealing the view.
func (m *ExpirationManager) Stop() error {
	// Stop all the pending expiration timers
	m.logger.Debug("expiration: stop triggered")
	defer m.logger.Debug("expiration: finished stopping")

	// Do this before stopping pending timers to avoid potential races with
	// expiring timers
	close(m.quitCh)

	m.pendingLock.Lock()
	for _, timer := range m.pending {
		timer.Stop()
	}
	m.pending = make(map[string]*time.Timer)
	m.pendingLock.Unlock()

	if m.inRestoreMode() {
		for {
			if !m.inRestoreMode() {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	return nil
}

// Revoke is used to revoke a secret named by the given LeaseID
func (m *ExpirationManager) Revoke(leaseID string) error {
	defer metrics.MeasureSince([]string{"expire", "revoke"}, time.Now())

	return m.revokeCommon(leaseID, false, false)
}

// revokeCommon does the heavy lifting. If force is true, we ignore a problem
// during revocation and still remove entries/index/lease timers
func (m *ExpirationManager) revokeCommon(leaseID string, force, skipToken bool) error {
	defer metrics.MeasureSince([]string{"expire", "revoke-common"}, time.Now())

	// Load the entry
	le, err := m.loadEntry(leaseID)
	if err != nil {
		return err
	}

	// If there is no entry, nothing to revoke
	if le == nil {
		return nil
	}

	// Revoke the entry
	if !skipToken || le.Auth == nil {
		if err := m.revokeEntry(le); err != nil {
			if !force {
				return err
			}

			if m.logger.IsWarn() {
				m.logger.Warn("revocation from the backend failed, but in force mode so ignoring", "error", err)
			}
		}
	}

	// Delete the entry
	if err := m.deleteEntry(leaseID); err != nil {
		return err
	}

	// Delete the secondary index, but only if it's a leased secret (not auth)
	if le.Secret != nil {
		if err := m.removeIndexByToken(le.ClientToken, le.LeaseID); err != nil {
			return err
		}
	}

	// Clear the expiration handler
	m.pendingLock.Lock()
	if timer, ok := m.pending[leaseID]; ok {
		timer.Stop()
		delete(m.pending, leaseID)
	}
	m.pendingLock.Unlock()
	return nil
}

// RevokeForce works similarly to RevokePrefix but continues in the case of a
// revocation error; this is mostly meant for recovery operations
func (m *ExpirationManager) RevokeForce(prefix string) error {
	defer metrics.MeasureSince([]string{"expire", "revoke-force"}, time.Now())

	return m.revokePrefixCommon(prefix, true)
}

// RevokePrefix is used to revoke all secrets with a given prefix.
// The prefix maps to that of the mount table to make this simpler
// to reason about.
func (m *ExpirationManager) RevokePrefix(prefix string) error {
	defer metrics.MeasureSince([]string{"expire", "revoke-prefix"}, time.Now())

	return m.revokePrefixCommon(prefix, false)
}

// RevokeByToken is used to revoke all the secrets issued with a given token.
// This is done by using the secondary index. It also removes the lease entry
// for the token itself. As a result it should *ONLY* ever be called from the
// token store's revokeSalted function.
func (m *ExpirationManager) RevokeByToken(te *TokenEntry) error {
	defer metrics.MeasureSince([]string{"expire", "revoke-by-token"}, time.Now())

	// Lookup the leases
	existing, err := m.lookupByToken(te.ID)
	if err != nil {
		return fmt.Errorf("failed to scan for leases: %v", err)
	}

	// Revoke all the keys
	for idx, leaseID := range existing {
		if err := m.revokeCommon(leaseID, false, false); err != nil {
			return fmt.Errorf("failed to revoke '%s' (%d / %d): %v",
				leaseID, idx+1, len(existing), err)
		}
	}

	if te.Path != "" {
		saltedID, err := m.tokenStore.SaltID(te.ID)
		if err != nil {
			return err
		}
		tokenLeaseID := path.Join(te.Path, saltedID)

		// We want to skip the revokeEntry call as that will call back into
		// revocation logic in the token store, which is what is running this
		// function in the first place -- it'd be a deadlock loop. Since the only
		// place that this function is called is revokeSalted in the token store,
		// we're already revoking the token, so we just want to clean up the lease.
		// This avoids spurious revocations later in the log when the timer runs
		// out, and eases up resource usage.
		return m.revokeCommon(tokenLeaseID, false, true)
	}

	return nil
}

func (m *ExpirationManager) revokePrefixCommon(prefix string, force bool) error {
	if m.inRestoreMode() {
		m.restoreRequestLock.Lock()
		defer m.restoreRequestLock.Unlock()
	}

	// Ensure there is a trailing slash
	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}

	// Accumulate existing leases
	sub := m.idView.SubView(prefix)
	existing, err := logical.CollectKeys(m.quitContext, sub)
	if err != nil {
		return fmt.Errorf("failed to scan for leases: %v", err)
	}

	// Revoke all the keys
	for idx, suffix := range existing {
		leaseID := prefix + suffix
		if err := m.revokeCommon(leaseID, force, false); err != nil {
			return fmt.Errorf("failed to revoke '%s' (%d / %d): %v",
				leaseID, idx+1, len(existing), err)
		}
	}
	return nil
}

// Renew is used to renew a secret using the given leaseID
// and a renew interval. The increment may be ignored.
func (m *ExpirationManager) Renew(leaseID string, increment time.Duration) (*logical.Response, error) {
	defer metrics.MeasureSince([]string{"expire", "renew"}, time.Now())

	// Load the entry
	le, err := m.loadEntry(leaseID)
	if err != nil {
		return nil, err
	}

	// Check if the lease is renewable
	if _, err := le.renewable(); err != nil {
		return nil, err
	}

	if le.Secret == nil {
		if le.Auth != nil {
			return logical.ErrorResponse("tokens cannot be renewed through this endpoint"), logical.ErrPermissionDenied
		}
		return logical.ErrorResponse("lease does not correspond to a secret"), nil
	}

	// Attempt to renew the entry
	resp, err := m.renewEntry(le, increment)
	if err != nil {
		return nil, err
	}

	// Fast-path if there is no lease
	if resp == nil || resp.Secret == nil || !resp.Secret.LeaseEnabled() {
		return resp, nil
	}

	// Validate the lease
	if err := resp.Secret.Validate(); err != nil {
		return nil, err
	}

	// Attach the LeaseID
	resp.Secret.LeaseID = leaseID

	// Update the lease entry
	le.Data = resp.Data
	le.Secret = resp.Secret
	le.ExpireTime = resp.Secret.ExpirationTime()
	le.LastRenewalTime = time.Now()
	if err := m.persistEntry(le); err != nil {
		return nil, err
	}

	// Update the expiration time
	m.updatePending(le, resp.Secret.LeaseTotal())

	// Return the response
	return resp, nil
}

// RestoreSaltedTokenCheck verifies that the token is not expired while running
// in restore mode.  If we are not in restore mode, the lease has already been
// restored or the lease still has time left, it returns true.
func (m *ExpirationManager) RestoreSaltedTokenCheck(source string, saltedID string) (bool, error) {
	defer metrics.MeasureSince([]string{"expire", "restore-token-check"}, time.Now())

	// Return immediately if we are not in restore mode, expiration manager is
	// already loaded
	if !m.inRestoreMode() {
		return true, nil
	}

	m.restoreModeLock.RLock()
	defer m.restoreModeLock.RUnlock()

	// Check again after we obtain the lock
	if !m.inRestoreMode() {
		return true, nil
	}

	leaseID := path.Join(source, saltedID)

	m.lockLease(leaseID)
	defer m.unlockLease(leaseID)

	le, err := m.loadEntryInternal(leaseID, true, true)
	if err != nil {
		return false, err
	}
	if le != nil && !le.ExpireTime.IsZero() {
		expires := le.ExpireTime.Sub(time.Now())
		if expires <= 0 {
			return false, nil
		}
	}

	return true, nil
}

// RenewToken is used to renew a token which does not need to
// invoke a logical backend.
func (m *ExpirationManager) RenewToken(req *logical.Request, source string, token string,
	increment time.Duration) (*logical.Response, error) {
	defer metrics.MeasureSince([]string{"expire", "renew-token"}, time.Now())

	// Compute the Lease ID
	saltedID, err := m.tokenStore.SaltID(token)
	if err != nil {
		return nil, err
	}
	leaseID := path.Join(source, saltedID)

	// Load the entry
	le, err := m.loadEntry(leaseID)
	if err != nil {
		return nil, err
	}

	// Check if the lease is renewable. Note that this also checks for a nil
	// lease and errors in that case as well.
	if _, err := le.renewable(); err != nil {
		return logical.ErrorResponse(err.Error()), logical.ErrInvalidRequest
	}

	// Attempt to renew the auth entry
	resp, err := m.renewAuthEntry(req, le, increment)
	if err != nil {
		return nil, err
	}

	if resp == nil {
		return nil, nil
	}

	if resp.IsError() {
		return &logical.Response{
			Data: resp.Data,
		}, nil
	}

	if resp.Auth == nil || !resp.Auth.LeaseEnabled() {
		return &logical.Response{
			Auth: resp.Auth,
		}, nil
	}

	sysView := m.router.MatchingSystemView(le.Path)
	if sysView == nil {
		return nil, fmt.Errorf("expiration: unable to retrieve system view from router")
	}

	retResp := &logical.Response{}
	switch {
	case resp.Auth.Period > time.Duration(0):
		// If it resp.Period is non-zero, use that as the TTL and override backend's
		// call on TTL modification, such as a TTL value determined by
		// framework.LeaseExtend call against the request. Also, cap period value to
		// the sys/mount max value.
		if resp.Auth.Period > sysView.MaxLeaseTTL() {
			retResp.AddWarning(fmt.Sprintf("Period of %d seconds is greater than current mount/system default of %d seconds, value will be truncated.", resp.Auth.TTL, sysView.MaxLeaseTTL()))
			resp.Auth.Period = sysView.MaxLeaseTTL()
		}
		resp.Auth.TTL = resp.Auth.Period
	case resp.Auth.TTL > time.Duration(0):
		// Cap TTL value to the sys/mount max value
		if resp.Auth.TTL > sysView.MaxLeaseTTL() {
			retResp.AddWarning(fmt.Sprintf("TTL of %d seconds is greater than current mount/system default of %d seconds, value will be truncated.", resp.Auth.TTL, sysView.MaxLeaseTTL()))
			resp.Auth.TTL = sysView.MaxLeaseTTL()
		}
	}

	// Attach the ClientToken
	resp.Auth.ClientToken = token
	resp.Auth.Increment = 0

	// Update the lease entry
	le.Auth = resp.Auth
	le.ExpireTime = resp.Auth.ExpirationTime()
	le.LastRenewalTime = time.Now()
	if err := m.persistEntry(le); err != nil {
		return nil, err
	}

	// Update the expiration time
	m.updatePending(le, resp.Auth.LeaseTotal())

	retResp.Auth = resp.Auth
	return retResp, nil
}

// Register is used to take a request and response with an associated
// lease. The secret gets assigned a LeaseID and the management of
// of lease is assumed by the expiration manager.
func (m *ExpirationManager) Register(req *logical.Request, resp *logical.Response) (id string, retErr error) {
	defer metrics.MeasureSince([]string{"expire", "register"}, time.Now())

	if req.ClientToken == "" {
		return "", fmt.Errorf("expiration: cannot register a lease with an empty client token")
	}

	// Ignore if there is no leased secret
	if resp == nil || resp.Secret == nil {
		return "", nil
	}

	// Validate the secret
	if err := resp.Secret.Validate(); err != nil {
		return "", err
	}

	// Create a lease entry
	leaseUUID, err := uuid.GenerateUUID()
	if err != nil {
		return "", err
	}

	leaseID := path.Join(req.Path, leaseUUID)

	defer func() {
		// If there is an error we want to rollback as much as possible (note
		// that errors here are ignored to do as much cleanup as we can). We
		// want to revoke a generated secret (since an error means we may not
		// be successfully tracking it), remove indexes, and delete the entry.
		if retErr != nil {
			revResp, err := m.router.Route(m.quitContext, logical.RevokeRequest(req.Path, resp.Secret, resp.Data))
			if err != nil {
				retErr = multierror.Append(retErr, errwrap.Wrapf("an additional internal error was encountered revoking the newly-generated secret: {{err}}", err))
			} else if revResp != nil && revResp.IsError() {
				retErr = multierror.Append(retErr, errwrap.Wrapf("an additional error was encountered revoking the newly-generated secret: {{err}}", revResp.Error()))
			}

			if err := m.deleteEntry(leaseID); err != nil {
				retErr = multierror.Append(retErr, errwrap.Wrapf("an additional error was encountered deleting any lease associated with the newly-generated secret: {{err}}", err))
			}

			if err := m.removeIndexByToken(req.ClientToken, leaseID); err != nil {
				retErr = multierror.Append(retErr, errwrap.Wrapf("an additional error was encountered removing lease indexes associated with the newly-generated secret: {{err}}", err))
			}
		}
	}()

	le := leaseEntry{
		LeaseID:     leaseID,
		ClientToken: req.ClientToken,
		Path:        req.Path,
		Data:        resp.Data,
		Secret:      resp.Secret,
		IssueTime:   time.Now(),
		ExpireTime:  resp.Secret.ExpirationTime(),
	}

	// Encode the entry
	if err := m.persistEntry(&le); err != nil {
		return "", err
	}

	// Maintain secondary index by token
	if err := m.createIndexByToken(le.ClientToken, le.LeaseID); err != nil {
		return "", err
	}

	// Setup revocation timer if there is a lease
	m.updatePending(&le, resp.Secret.LeaseTotal())

	// Done
	return le.LeaseID, nil
}

// RegisterAuth is used to take an Auth response with an associated lease.
// The token does not get a LeaseID, but the lease management is handled by
// the expiration manager.
func (m *ExpirationManager) RegisterAuth(source string, auth *logical.Auth) error {
	defer metrics.MeasureSince([]string{"expire", "register-auth"}, time.Now())

	if auth.ClientToken == "" {
		return fmt.Errorf("expiration: cannot register an auth lease with an empty token")
	}

	if strings.Contains(source, "..") {
		return fmt.Errorf("expiration: %s", consts.ErrPathContainsParentReferences)
	}

	saltedID, err := m.tokenStore.SaltID(auth.ClientToken)
	if err != nil {
		return err
	}

	// If it resp.Period is non-zero, override the TTL value determined
	// by the backend.
	if auth.Period > time.Duration(0) {
		auth.TTL = auth.Period
	}

	// Create a lease entry
	le := leaseEntry{
		LeaseID:     path.Join(source, saltedID),
		ClientToken: auth.ClientToken,
		Auth:        auth,
		Path:        source,
		IssueTime:   time.Now(),
		ExpireTime:  auth.ExpirationTime(),
	}

	// Encode the entry
	if err := m.persistEntry(&le); err != nil {
		return err
	}

	// Setup revocation timer
	m.updatePending(&le, auth.LeaseTotal())
	return nil
}

// FetchLeaseTimesByToken is a helper function to use token values to compute
// the leaseID, rather than pushing that logic back into the token store.
func (m *ExpirationManager) FetchLeaseTimesByToken(source, token string) (*leaseEntry, error) {
	defer metrics.MeasureSince([]string{"expire", "fetch-lease-times-by-token"}, time.Now())

	// Compute the Lease ID
	saltedID, err := m.tokenStore.SaltID(token)
	if err != nil {
		return nil, err
	}
	leaseID := path.Join(source, saltedID)
	return m.FetchLeaseTimes(leaseID)
}

// FetchLeaseTimes is used to fetch the issue time, expiration time, and last
// renewed time of a lease entry. It returns a leaseEntry itself, but with only
// those values copied over.
func (m *ExpirationManager) FetchLeaseTimes(leaseID string) (*leaseEntry, error) {
	defer metrics.MeasureSince([]string{"expire", "fetch-lease-times"}, time.Now())

	// Load the entry
	le, err := m.loadEntry(leaseID)
	if err != nil {
		return nil, err
	}
	if le == nil {
		return nil, nil
	}

	ret := &leaseEntry{
		IssueTime:       le.IssueTime,
		ExpireTime:      le.ExpireTime,
		LastRenewalTime: le.LastRenewalTime,
	}
	if le.Secret != nil {
		ret.Secret = &logical.Secret{}
		ret.Secret.Renewable = le.Secret.Renewable
		ret.Secret.TTL = le.Secret.TTL
	}
	if le.Auth != nil {
		ret.Auth = &logical.Auth{}
		ret.Auth.Renewable = le.Auth.Renewable
		ret.Auth.TTL = le.Auth.TTL
	}

	return ret, nil
}

// updatePending is used to update a pending invocation for a lease
func (m *ExpirationManager) updatePending(le *leaseEntry, leaseTotal time.Duration) {
	m.pendingLock.Lock()
	defer m.pendingLock.Unlock()

	// Check for an existing timer
	timer, ok := m.pending[le.LeaseID]

	// If there is no expiry time, don't do anything
	if le.ExpireTime.IsZero() {
		// if the timer happened to exist, stop the time and delete it from the
		// pending timers.
		if ok {
			timer.Stop()
			delete(m.pending, le.LeaseID)
		}
		return
	}

	// Create entry if it does not exist
	if !ok {
		timer := time.AfterFunc(leaseTotal, func() {
			m.expireID(le.LeaseID)
		})
		m.pending[le.LeaseID] = timer
		return
	}

	// Extend the timer by the lease total
	timer.Reset(leaseTotal)
}

// expireID is invoked when a given ID is expired
func (m *ExpirationManager) expireID(leaseID string) {
	// Clear from the pending expiration
	m.pendingLock.Lock()
	delete(m.pending, leaseID)
	m.pendingLock.Unlock()

	for attempt := uint(0); attempt < maxRevokeAttempts; attempt++ {
		select {
		case <-m.quitCh:
			m.logger.Error("expiration: shutting down, not attempting further revocation of lease", "lease_id", leaseID)
			return
		default:
		}

		m.coreStateLock.RLock()
		if m.quitContext.Err() == context.Canceled {
			m.logger.Error("expiration: core context canceled, not attempting further revocation of lease", "lease_id", leaseID)
			m.coreStateLock.RUnlock()
			return
		}

		err := m.Revoke(leaseID)
		if err == nil {
			if m.logger.IsInfo() {
				m.logger.Info("expiration: revoked lease", "lease_id", leaseID)
			}
			m.coreStateLock.RUnlock()
			return
		}

		m.coreStateLock.RUnlock()
		m.logger.Error("expiration: failed to revoke lease", "lease_id", leaseID, "error", err)
		time.Sleep((1 << attempt) * revokeRetryBase)
	}
	m.logger.Error("expiration: maximum revoke attempts reached", "lease_id", leaseID)
}

// revokeEntry is used to attempt revocation of an internal entry
func (m *ExpirationManager) revokeEntry(le *leaseEntry) error {
	// Revocation of login tokens is special since we can by-pass the
	// backend and directly interact with the token store
	if le.Auth != nil {
		if err := m.tokenStore.RevokeTree(m.quitContext, le.ClientToken); err != nil {
			return fmt.Errorf("failed to revoke token: %v", err)
		}

		return nil
	}

	// Handle standard revocation via backends
	resp, err := m.router.Route(m.quitContext, logical.RevokeRequest(le.Path, le.Secret, le.Data))
	if err != nil || (resp != nil && resp.IsError()) {
		return fmt.Errorf("failed to revoke entry: resp:%#v err:%s", resp, err)
	}
	return nil
}

// renewEntry is used to attempt renew of an internal entry
func (m *ExpirationManager) renewEntry(le *leaseEntry, increment time.Duration) (*logical.Response, error) {
	secret := *le.Secret
	secret.IssueTime = le.IssueTime
	secret.Increment = increment
	secret.LeaseID = ""

	req := logical.RenewRequest(le.Path, &secret, le.Data)
	resp, err := m.router.Route(m.quitContext, req)
	if err != nil || (resp != nil && resp.IsError()) {
		return nil, fmt.Errorf("failed to renew entry: resp:%#v err:%s", resp, err)
	}
	return resp, nil
}

// renewAuthEntry is used to attempt renew of an auth entry. Only the token
// store should get the actual token ID intact.
func (m *ExpirationManager) renewAuthEntry(req *logical.Request, le *leaseEntry, increment time.Duration) (*logical.Response, error) {
	auth := *le.Auth
	auth.IssueTime = le.IssueTime
	auth.Increment = increment
	if strings.HasPrefix(le.Path, "auth/token/") {
		auth.ClientToken = le.ClientToken
	} else {
		auth.ClientToken = ""
	}

	authReq := logical.RenewAuthRequest(le.Path, &auth, nil)
	authReq.Connection = req.Connection
	resp, err := m.router.Route(m.quitContext, authReq)
	if err != nil {
		return nil, fmt.Errorf("failed to renew entry: %v", err)
	}
	return resp, nil
}

// loadEntry is used to read a lease entry
func (m *ExpirationManager) loadEntry(leaseID string) (*leaseEntry, error) {
	// Take out the lease locks after we ensure we are in restore mode
	restoreMode := m.inRestoreMode()
	if restoreMode {
		m.restoreModeLock.RLock()
		defer m.restoreModeLock.RUnlock()

		restoreMode = m.inRestoreMode()
		if restoreMode {
			m.lockLease(leaseID)
			defer m.unlockLease(leaseID)
		}
	}
	return m.loadEntryInternal(leaseID, restoreMode, true)
}

// loadEntryInternal is used when you need to load an entry but also need to
// control the lifecycle of the restoreLock
func (m *ExpirationManager) loadEntryInternal(leaseID string, restoreMode bool, checkRestored bool) (*leaseEntry, error) {
	out, err := m.idView.Get(m.quitContext, leaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to read lease entry: %v", err)
	}
	if out == nil {
		return nil, nil
	}
	le, err := decodeLeaseEntry(out.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to decode lease entry: %v", err)
	}

	if restoreMode {
		if checkRestored {
			// If we have already loaded this lease, we don't need to update on
			// load. In the case of renewal and revocation, updatePending will be
			// done after making the appropriate modifications to the lease.
			if _, ok := m.restoreLoaded.Load(leaseID); ok {
				return le, nil
			}
		}

		// Update the cache of restored leases, either synchronously or through
		// the lazy loaded restore process
		m.restoreLoaded.Store(le.LeaseID, struct{}{})

		// Setup revocation timer
		m.updatePending(le, le.ExpireTime.Sub(time.Now()))
	}
	return le, nil
}

// persistEntry is used to persist a lease entry
func (m *ExpirationManager) persistEntry(le *leaseEntry) error {
	// Encode the entry
	buf, err := le.encode()
	if err != nil {
		return fmt.Errorf("failed to encode lease entry: %v", err)
	}

	// Write out to the view
	ent := logical.StorageEntry{
		Key:   le.LeaseID,
		Value: buf,
	}
	if le.Auth != nil && len(le.Auth.Policies) == 1 && le.Auth.Policies[0] == "root" {
		ent.SealWrap = true
	}
	if err := m.idView.Put(m.quitContext, &ent); err != nil {
		return fmt.Errorf("failed to persist lease entry: %v", err)
	}
	return nil
}

// deleteEntry is used to delete a lease entry
func (m *ExpirationManager) deleteEntry(leaseID string) error {
	if err := m.idView.Delete(m.quitContext, leaseID); err != nil {
		return fmt.Errorf("failed to delete lease entry: %v", err)
	}
	return nil
}

// createIndexByToken creates a secondary index from the token to a lease entry
func (m *ExpirationManager) createIndexByToken(token, leaseID string) error {
	saltedID, err := m.tokenStore.SaltID(token)
	if err != nil {
		return err
	}

	leaseSaltedID, err := m.tokenStore.SaltID(leaseID)
	if err != nil {
		return err
	}

	ent := logical.StorageEntry{
		Key:   saltedID + "/" + leaseSaltedID,
		Value: []byte(leaseID),
	}
	if err := m.tokenView.Put(m.quitContext, &ent); err != nil {
		return fmt.Errorf("failed to persist lease index entry: %v", err)
	}
	return nil
}

// indexByToken looks up the secondary index from the token to a lease entry
func (m *ExpirationManager) indexByToken(token, leaseID string) (*logical.StorageEntry, error) {
	saltedID, err := m.tokenStore.SaltID(token)
	if err != nil {
		return nil, err
	}

	leaseSaltedID, err := m.tokenStore.SaltID(leaseID)
	if err != nil {
		return nil, err
	}

	key := saltedID + "/" + leaseSaltedID
	entry, err := m.tokenView.Get(m.quitContext, key)
	if err != nil {
		return nil, fmt.Errorf("failed to look up secondary index entry")
	}
	return entry, nil
}

// removeIndexByToken removes the secondary index from the token to a lease entry
func (m *ExpirationManager) removeIndexByToken(token, leaseID string) error {
	saltedID, err := m.tokenStore.SaltID(token)
	if err != nil {
		return err
	}

	leaseSaltedID, err := m.tokenStore.SaltID(leaseID)
	if err != nil {
		return err
	}

	key := saltedID + "/" + leaseSaltedID
	if err := m.tokenView.Delete(m.quitContext, key); err != nil {
		return fmt.Errorf("failed to delete lease index entry: %v", err)
	}
	return nil
}

// lookupByToken is used to lookup all the leaseID's via the
func (m *ExpirationManager) lookupByToken(token string) ([]string, error) {
	saltedID, err := m.tokenStore.SaltID(token)
	if err != nil {
		return nil, err
	}

	// Scan via the index for sub-leases
	prefix := saltedID + "/"
	subKeys, err := m.tokenView.List(m.quitContext, prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list leases: %v", err)
	}

	// Read each index entry
	leaseIDs := make([]string, 0, len(subKeys))
	for _, sub := range subKeys {
		out, err := m.tokenView.Get(m.quitContext, prefix+sub)
		if err != nil {
			return nil, fmt.Errorf("failed to read lease index: %v", err)
		}
		if out == nil {
			continue
		}
		leaseIDs = append(leaseIDs, string(out.Value))
	}
	return leaseIDs, nil
}

// emitMetrics is invoked periodically to emit statistics
func (m *ExpirationManager) emitMetrics() {
	m.pendingLock.RLock()
	num := len(m.pending)
	m.pendingLock.RUnlock()
	metrics.SetGauge([]string{"expire", "num_leases"}, float32(num))
	// Check if lease count is greater than the threshold
	if num > maxLeaseThreshold {
		if atomic.LoadUint32(&m.leaseCheckCounter) > 59 {
			m.logger.Warn("expiration: lease count exceeds warning lease threshold")
			atomic.StoreUint32(&m.leaseCheckCounter, 0)
		} else {
			atomic.AddUint32(&m.leaseCheckCounter, 1)
		}
	}
}

// leaseEntry is used to structure the values the expiration
// manager stores. This is used to handle renew and revocation.
type leaseEntry struct {
	LeaseID         string                 `json:"lease_id"`
	ClientToken     string                 `json:"client_token"`
	Path            string                 `json:"path"`
	Data            map[string]interface{} `json:"data"`
	Secret          *logical.Secret        `json:"secret"`
	Auth            *logical.Auth          `json:"auth"`
	IssueTime       time.Time              `json:"issue_time"`
	ExpireTime      time.Time              `json:"expire_time"`
	LastRenewalTime time.Time              `json:"last_renewal_time"`
}

// encode is used to JSON encode the lease entry
func (le *leaseEntry) encode() ([]byte, error) {
	return json.Marshal(le)
}

func (le *leaseEntry) renewable() (bool, error) {
	var err error
	switch {
	// If there is no entry, cannot review
	case le == nil || le.ExpireTime.IsZero():
		err = fmt.Errorf("lease not found or lease is not renewable")
	// Determine if the lease is expired
	case le.ExpireTime.Before(time.Now()):
		err = fmt.Errorf("lease expired")
	// Determine if the lease is renewable
	case le.Secret != nil && !le.Secret.Renewable:
		err = fmt.Errorf("lease is not renewable")
	case le.Auth != nil && !le.Auth.Renewable:
		err = fmt.Errorf("lease is not renewable")
	}

	if err != nil {
		return false, err
	}
	return true, nil
}

func (le *leaseEntry) ttl() int64 {
	return int64(le.ExpireTime.Sub(time.Now().Round(time.Second)).Seconds())
}

// decodeLeaseEntry is used to reverse encode and return a new entry
func decodeLeaseEntry(buf []byte) (*leaseEntry, error) {
	out := new(leaseEntry)
	return out, jsonutil.DecodeJSON(buf, out)
}
