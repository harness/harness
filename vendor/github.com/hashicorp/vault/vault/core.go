package vault

import (
	"context"
	"crypto/ecdsa"
	"crypto/subtle"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/armon/go-metrics"
	log "github.com/mgutz/logxi/v1"

	"google.golang.org/grpc"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/vault/audit"
	"github.com/hashicorp/vault/helper/consts"
	"github.com/hashicorp/vault/helper/errutil"
	"github.com/hashicorp/vault/helper/identity"
	"github.com/hashicorp/vault/helper/jsonutil"
	"github.com/hashicorp/vault/helper/logformat"
	"github.com/hashicorp/vault/helper/mlock"
	"github.com/hashicorp/vault/helper/reload"
	"github.com/hashicorp/vault/helper/tlsutil"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/physical"
	"github.com/hashicorp/vault/shamir"
	cache "github.com/patrickmn/go-cache"
)

const (
	// coreLockPath is the path used to acquire a coordinating lock
	// for a highly-available deploy.
	coreLockPath = "core/lock"

	// The poison pill is used as a check during certain scenarios to indicate
	// to standby nodes that they should seal
	poisonPillPath = "core/poison-pill"

	// coreLeaderPrefix is the prefix used for the UUID that contains
	// the currently elected leader.
	coreLeaderPrefix = "core/leader/"

	// knownPrimaryAddrsPrefix is used to store last-known cluster address
	// information for primaries
	knownPrimaryAddrsPrefix = "core/primary-addrs/"

	// lockRetryInterval is the interval we re-attempt to acquire the
	// HA lock if an error is encountered
	lockRetryInterval = 10 * time.Second

	// leaderCheckInterval is how often a standby checks for a new leader
	leaderCheckInterval = 2500 * time.Millisecond

	// keyRotateCheckInterval is how often a standby checks for a key
	// rotation taking place.
	keyRotateCheckInterval = 30 * time.Second

	// keyRotateGracePeriod is how long we allow an upgrade path
	// for standby instances before we delete the upgrade keys
	keyRotateGracePeriod = 2 * time.Minute

	// leaderPrefixCleanDelay is how long to wait between deletions
	// of orphaned leader keys, to prevent slamming the backend.
	leaderPrefixCleanDelay = 200 * time.Millisecond

	// coreKeyringCanaryPath is used as a canary to indicate to replicated
	// clusters that they need to perform a rekey operation synchronously; this
	// isn't keyring-canary to avoid ignoring it when ignoring core/keyring
	coreKeyringCanaryPath = "core/canary-keyring"
)

var (
	// ErrAlreadyInit is returned if the core is already
	// initialized. This prevents a re-initialization.
	ErrAlreadyInit = errors.New("Vault is already initialized")

	// ErrNotInit is returned if a non-initialized barrier
	// is attempted to be unsealed.
	ErrNotInit = errors.New("Vault is not initialized")

	// ErrInternalError is returned when we don't want to leak
	// any information about an internal error
	ErrInternalError = errors.New("internal error")

	// ErrHANotEnabled is returned if the operation only makes sense
	// in an HA setting
	ErrHANotEnabled = errors.New("Vault is not configured for highly-available mode")

	// manualStepDownSleepPeriod is how long to sleep after a user-initiated
	// step down of the active node, to prevent instantly regrabbing the lock.
	// It's var not const so that tests can manipulate it.
	manualStepDownSleepPeriod = 10 * time.Second

	// Functions only in the Enterprise version
	enterprisePostUnseal = enterprisePostUnsealImpl
	enterprisePreSeal    = enterprisePreSealImpl
	startReplication     = startReplicationImpl
	stopReplication      = stopReplicationImpl
	LastRemoteWAL        = lastRemoteWALImpl
)

// NonFatalError is an error that can be returned during NewCore that should be
// displayed but not cause a program exit
type NonFatalError struct {
	Err error
}

func (e *NonFatalError) WrappedErrors() []error {
	return []error{e.Err}
}

func (e *NonFatalError) Error() string {
	return e.Err.Error()
}

// ErrInvalidKey is returned if there is a user-based error with a provided
// unseal key. This will be shown to the user, so should not contain
// information that is sensitive.
type ErrInvalidKey struct {
	Reason string
}

func (e *ErrInvalidKey) Error() string {
	return fmt.Sprintf("invalid key: %v", e.Reason)
}

type activeAdvertisement struct {
	RedirectAddr     string            `json:"redirect_addr"`
	ClusterAddr      string            `json:"cluster_addr,omitempty"`
	ClusterCert      []byte            `json:"cluster_cert,omitempty"`
	ClusterKeyParams *clusterKeyParams `json:"cluster_key_params,omitempty"`
}

type unlockInformation struct {
	Parts [][]byte
	Nonce string
}

// Core is used as the central manager of Vault activity. It is the primary point of
// interface for API handlers and is responsible for managing the logical and physical
// backends, router, security barrier, and audit trails.
type Core struct {
	// N.B.: This is used to populate a dev token down replication, as
	// otherwise, after replication is started, a dev would have to go through
	// the generate-root process simply to talk to the new follower cluster.
	devToken string

	// HABackend may be available depending on the physical backend
	ha physical.HABackend

	// redirectAddr is the address we advertise as leader if held
	redirectAddr string

	// clusterAddr is the address we use for clustering
	clusterAddr string

	// physical backend is the un-trusted backend with durable data
	physical physical.Backend

	// Our Seal, for seal configuration information
	seal Seal

	// barrier is the security barrier wrapping the physical backend
	barrier SecurityBarrier

	// router is responsible for managing the mount points for logical backends.
	router *Router

	// logicalBackends is the mapping of backends to use for this core
	logicalBackends map[string]logical.Factory

	// credentialBackends is the mapping of backends to use for this core
	credentialBackends map[string]logical.Factory

	// auditBackends is the mapping of backends to use for this core
	auditBackends map[string]audit.Factory

	// stateLock protects mutable state
	stateLock sync.RWMutex
	sealed    bool

	standby          bool
	standbyDoneCh    chan struct{}
	standbyStopCh    chan struct{}
	manualStepDownCh chan struct{}

	// unlockInfo has the keys provided to Unseal until the threshold number of parts is available, as well as the operation nonce
	unlockInfo *unlockInformation

	// generateRootProgress holds the shares until we reach enough
	// to verify the master key
	generateRootConfig   *GenerateRootConfig
	generateRootProgress [][]byte
	generateRootLock     sync.Mutex

	// These variables holds the config and shares we have until we reach
	// enough to verify the appropriate master key. Note that the same lock is
	// used; this isn't time-critical so this shouldn't be a problem.
	barrierRekeyConfig    *SealConfig
	barrierRekeyProgress  [][]byte
	recoveryRekeyConfig   *SealConfig
	recoveryRekeyProgress [][]byte
	rekeyLock             sync.RWMutex

	// mounts is loaded after unseal since it is a protected
	// configuration
	mounts *MountTable

	// mountsLock is used to ensure that the mounts table does not
	// change underneath a calling function
	mountsLock sync.RWMutex

	// auth is loaded after unseal since it is a protected
	// configuration
	auth *MountTable

	// authLock is used to ensure that the auth table does not
	// change underneath a calling function
	authLock sync.RWMutex

	// audit is loaded after unseal since it is a protected
	// configuration
	audit *MountTable

	// auditLock is used to ensure that the audit table does not
	// change underneath a calling function
	auditLock sync.RWMutex

	// auditBroker is used to ingest the audit events and fan
	// out into the configured audit backends
	auditBroker *AuditBroker

	// auditedHeaders is used to configure which http headers
	// can be output in the audit logs
	auditedHeaders *AuditedHeadersConfig

	// systemBackend is the backend which is used to manage internal operations
	systemBackend *SystemBackend

	// systemBarrierView is the barrier view for the system backend
	systemBarrierView *BarrierView

	// expiration manager is used for managing LeaseIDs,
	// renewal, expiration and revocation
	expiration *ExpirationManager

	// rollback manager is used to run rollbacks periodically
	rollback *RollbackManager

	// policy store is used to manage named ACL policies
	policyStore *PolicyStore

	// token store is used to manage authentication tokens
	tokenStore *TokenStore

	// identityStore is used to manage client entities
	identityStore *IdentityStore

	// metricsCh is used to stop the metrics streaming
	metricsCh chan struct{}

	// metricsMutex is used to prevent a race condition between
	// metrics emission and sealing leading to a nil pointer
	metricsMutex sync.Mutex

	defaultLeaseTTL time.Duration
	maxLeaseTTL     time.Duration

	logger log.Logger

	// cachingDisabled indicates whether caches are disabled
	cachingDisabled bool
	// Cache stores the actual cache; we always have this but may bypass it if
	// disabled
	physicalCache physical.ToggleablePurgemonster

	// reloadFuncs is a map containing reload functions
	reloadFuncs map[string][]reload.ReloadFunc

	// reloadFuncsLock controls access to the funcs
	reloadFuncsLock sync.RWMutex

	// wrappingJWTKey is the key used for generating JWTs containing response
	// wrapping information
	wrappingJWTKey *ecdsa.PrivateKey

	//
	// Cluster information
	//
	// Name
	clusterName string
	// Specific cipher suites to use for clustering, if any
	clusterCipherSuites []uint16
	// Used to modify cluster parameters
	clusterParamsLock sync.RWMutex
	// The private key stored in the barrier used for establishing
	// mutually-authenticated connections between Vault cluster members
	localClusterPrivateKey *atomic.Value
	// The local cluster cert
	localClusterCert *atomic.Value
	// The parsed form of the local cluster cert
	localClusterParsedCert *atomic.Value
	// The TCP addresses we should use for clustering
	clusterListenerAddrs []*net.TCPAddr
	// The handler to use for request forwarding
	clusterHandler http.Handler
	// Tracks whether cluster listeners are running, e.g. it's safe to send a
	// shutdown down the channel
	clusterListenersRunning bool
	// Shutdown channel for the cluster listeners
	clusterListenerShutdownCh chan struct{}
	// Shutdown success channel. We need this to be done serially to ensure
	// that binds are removed before they might be reinstated.
	clusterListenerShutdownSuccessCh chan struct{}
	// Write lock used to ensure that we don't have multiple connections adjust
	// this value at the same time
	requestForwardingConnectionLock sync.RWMutex
	// Most recent leader UUID. Used to avoid repeatedly JSON parsing the same
	// values.
	clusterLeaderUUID string
	// Most recent leader redirect addr
	clusterLeaderRedirectAddr string
	// Most recent leader cluster addr
	clusterLeaderClusterAddr string
	// Lock for the cluster leader values
	clusterLeaderParamsLock sync.RWMutex
	// Info on cluster members
	clusterPeerClusterAddrsCache *cache.Cache
	// Stores whether we currently have a server running
	rpcServerActive *uint32
	// The context for the client
	rpcClientConnContext context.Context
	// The function for canceling the client connection
	rpcClientConnCancelFunc context.CancelFunc
	// The grpc ClientConn for RPC calls
	rpcClientConn *grpc.ClientConn
	// The grpc forwarding client
	rpcForwardingClient *forwardingClient

	// CORS Information
	corsConfig *CORSConfig

	// The active set of upstream cluster addresses; stored via the Echo
	// mechanism, loaded by the balancer
	atomicPrimaryClusterAddrs *atomic.Value

	atomicPrimaryFailoverAddrs *atomic.Value
	// replicationState keeps the current replication state cached for quick
	// lookup; activeNodeReplicationState stores the active value on standbys
	replicationState           *uint32
	activeNodeReplicationState *uint32

	// uiEnabled indicates whether Vault Web UI is enabled or not
	uiEnabled bool

	// rawEnabled indicates whether the Raw endpoint is enabled
	rawEnabled bool

	// pluginDirectory is the location vault will look for plugin binaries
	pluginDirectory string

	// pluginCatalog is used to manage plugin configurations
	pluginCatalog *PluginCatalog

	enableMlock bool

	// This can be used to trigger operations to stop running when Vault is
	// going to be shut down, stepped down, or sealed
	activeContext           context.Context
	activeContextCancelFunc context.CancelFunc

	// Stores the sealunwrapper for downgrade needs
	sealUnwrapper physical.Backend
}

// CoreConfig is used to parameterize a core
type CoreConfig struct {
	DevToken string `json:"dev_token" structs:"dev_token" mapstructure:"dev_token"`

	LogicalBackends map[string]logical.Factory `json:"logical_backends" structs:"logical_backends" mapstructure:"logical_backends"`

	CredentialBackends map[string]logical.Factory `json:"credential_backends" structs:"credential_backends" mapstructure:"credential_backends"`

	AuditBackends map[string]audit.Factory `json:"audit_backends" structs:"audit_backends" mapstructure:"audit_backends"`

	Physical physical.Backend `json:"physical" structs:"physical" mapstructure:"physical"`

	// May be nil, which disables HA operations
	HAPhysical physical.HABackend `json:"ha_physical" structs:"ha_physical" mapstructure:"ha_physical"`

	Seal Seal `json:"seal" structs:"seal" mapstructure:"seal"`

	Logger log.Logger `json:"logger" structs:"logger" mapstructure:"logger"`

	// Disables the LRU cache on the physical backend
	DisableCache bool `json:"disable_cache" structs:"disable_cache" mapstructure:"disable_cache"`

	// Disables mlock syscall
	DisableMlock bool `json:"disable_mlock" structs:"disable_mlock" mapstructure:"disable_mlock"`

	// Custom cache size for the LRU cache on the physical backend, or zero for default
	CacheSize int `json:"cache_size" structs:"cache_size" mapstructure:"cache_size"`

	// Set as the leader address for HA
	RedirectAddr string `json:"redirect_addr" structs:"redirect_addr" mapstructure:"redirect_addr"`

	// Set as the cluster address for HA
	ClusterAddr string `json:"cluster_addr" structs:"cluster_addr" mapstructure:"cluster_addr"`

	DefaultLeaseTTL time.Duration `json:"default_lease_ttl" structs:"default_lease_ttl" mapstructure:"default_lease_ttl"`

	MaxLeaseTTL time.Duration `json:"max_lease_ttl" structs:"max_lease_ttl" mapstructure:"max_lease_ttl"`

	ClusterName string `json:"cluster_name" structs:"cluster_name" mapstructure:"cluster_name"`

	ClusterCipherSuites string `json:"cluster_cipher_suites" structs:"cluster_cipher_suites" mapstructure:"cluster_cipher_suites"`

	EnableUI bool `json:"ui" structs:"ui" mapstructure:"ui"`

	// Enable the raw endpoint
	EnableRaw bool `json:"enable_raw" structs:"enable_raw" mapstructure:"enable_raw"`

	PluginDirectory string `json:"plugin_directory" structs:"plugin_directory" mapstructure:"plugin_directory"`

	ReloadFuncs     *map[string][]reload.ReloadFunc
	ReloadFuncsLock *sync.RWMutex
}

// NewCore is used to construct a new core
func NewCore(conf *CoreConfig) (*Core, error) {
	if conf.HAPhysical != nil && conf.HAPhysical.HAEnabled() {
		if conf.RedirectAddr == "" {
			return nil, fmt.Errorf("missing API address, please set in configuration or via environment")
		}
	}

	if conf.DefaultLeaseTTL == 0 {
		conf.DefaultLeaseTTL = defaultLeaseTTL
	}
	if conf.MaxLeaseTTL == 0 {
		conf.MaxLeaseTTL = maxLeaseTTL
	}
	if conf.DefaultLeaseTTL > conf.MaxLeaseTTL {
		return nil, fmt.Errorf("cannot have DefaultLeaseTTL larger than MaxLeaseTTL")
	}

	// Validate the advertise addr if its given to us
	if conf.RedirectAddr != "" {
		u, err := url.Parse(conf.RedirectAddr)
		if err != nil {
			return nil, fmt.Errorf("redirect address is not valid url: %s", err)
		}

		if u.Scheme == "" {
			return nil, fmt.Errorf("redirect address must include scheme (ex. 'http')")
		}
	}

	// Make a default logger if not provided
	if conf.Logger == nil {
		conf.Logger = logformat.NewVaultLogger(log.LevelTrace)
	}

	// Setup the core
	c := &Core{
		devToken:                         conf.DevToken,
		physical:                         conf.Physical,
		redirectAddr:                     conf.RedirectAddr,
		clusterAddr:                      conf.ClusterAddr,
		seal:                             conf.Seal,
		router:                           NewRouter(),
		sealed:                           true,
		standby:                          true,
		logger:                           conf.Logger,
		defaultLeaseTTL:                  conf.DefaultLeaseTTL,
		maxLeaseTTL:                      conf.MaxLeaseTTL,
		cachingDisabled:                  conf.DisableCache,
		clusterName:                      conf.ClusterName,
		clusterListenerShutdownCh:        make(chan struct{}),
		clusterListenerShutdownSuccessCh: make(chan struct{}),
		clusterPeerClusterAddrsCache:     cache.New(3*HeartbeatInterval, time.Second),
		enableMlock:                      !conf.DisableMlock,
		rawEnabled:                       conf.EnableRaw,
		replicationState:                 new(uint32),
		rpcServerActive:                  new(uint32),
		atomicPrimaryClusterAddrs:        new(atomic.Value),
		atomicPrimaryFailoverAddrs:       new(atomic.Value),
		localClusterPrivateKey:           new(atomic.Value),
		localClusterCert:                 new(atomic.Value),
		localClusterParsedCert:           new(atomic.Value),
		activeNodeReplicationState:       new(uint32),
	}

	atomic.StoreUint32(c.replicationState, uint32(consts.ReplicationDRDisabled|consts.ReplicationPerformanceDisabled))
	c.localClusterCert.Store(([]byte)(nil))
	c.localClusterParsedCert.Store((*x509.Certificate)(nil))
	c.localClusterPrivateKey.Store((*ecdsa.PrivateKey)(nil))

	if conf.ClusterCipherSuites != "" {
		suites, err := tlsutil.ParseCiphers(conf.ClusterCipherSuites)
		if err != nil {
			return nil, errwrap.Wrapf("error parsing cluster cipher suites: {{err}}", err)
		}
		c.clusterCipherSuites = suites
	}

	// Load CORS config and provide a value for the core field.
	c.corsConfig = &CORSConfig{core: c}

	phys := conf.Physical
	_, txnOK := conf.Physical.(physical.Transactional)
	if c.seal == nil {
		c.seal = NewDefaultSeal()
	}
	c.seal.SetCore(c)

	c.sealUnwrapper = NewSealUnwrapper(phys, conf.Logger)

	var ok bool

	// Wrap the physical backend in a cache layer if enabled
	if txnOK {
		c.physical = physical.NewTransactionalCache(c.sealUnwrapper, conf.CacheSize, conf.Logger)
	} else {
		c.physical = physical.NewCache(c.sealUnwrapper, conf.CacheSize, conf.Logger)
	}
	c.physicalCache = c.physical.(physical.ToggleablePurgemonster)

	if !conf.DisableMlock {
		// Ensure our memory usage is locked into physical RAM
		if err := mlock.LockMemory(); err != nil {
			return nil, fmt.Errorf(
				"Failed to lock memory: %v\n\n"+
					"This usually means that the mlock syscall is not available.\n"+
					"Vault uses mlock to prevent memory from being swapped to\n"+
					"disk. This requires root privileges as well as a machine\n"+
					"that supports mlock. Please enable mlock on your system or\n"+
					"disable Vault from using it. To disable Vault from using it,\n"+
					"set the `disable_mlock` configuration option in your configuration\n"+
					"file.",
				err)
		}
	}

	var err error
	if conf.PluginDirectory != "" {
		c.pluginDirectory, err = filepath.Abs(conf.PluginDirectory)
		if err != nil {
			return nil, fmt.Errorf("core setup failed, could not verify plugin directory: %v", err)
		}
	}

	// Construct a new AES-GCM barrier
	c.barrier, err = NewAESGCMBarrier(c.physical)
	if err != nil {
		return nil, fmt.Errorf("barrier setup failed: %v", err)
	}

	if conf.HAPhysical != nil && conf.HAPhysical.HAEnabled() {
		c.ha = conf.HAPhysical
	}

	// We create the funcs here, then populate the given config with it so that
	// the caller can share state
	conf.ReloadFuncsLock = &c.reloadFuncsLock
	c.reloadFuncsLock.Lock()
	c.reloadFuncs = make(map[string][]reload.ReloadFunc)
	c.reloadFuncsLock.Unlock()
	conf.ReloadFuncs = &c.reloadFuncs

	// Setup the backends
	logicalBackends := make(map[string]logical.Factory)
	for k, f := range conf.LogicalBackends {
		logicalBackends[k] = f
	}
	_, ok = logicalBackends["kv"]
	if !ok {
		logicalBackends["kv"] = PassthroughBackendFactory
	}
	logicalBackends["cubbyhole"] = CubbyholeBackendFactory
	logicalBackends["system"] = func(ctx context.Context, config *logical.BackendConfig) (logical.Backend, error) {
		b := NewSystemBackend(c)
		if err := b.Setup(ctx, config); err != nil {
			return nil, err
		}
		return b, nil
	}

	logicalBackends["identity"] = func(ctx context.Context, config *logical.BackendConfig) (logical.Backend, error) {
		return NewIdentityStore(ctx, c, config)
	}

	c.logicalBackends = logicalBackends

	credentialBackends := make(map[string]logical.Factory)
	for k, f := range conf.CredentialBackends {
		credentialBackends[k] = f
	}
	credentialBackends["token"] = func(ctx context.Context, config *logical.BackendConfig) (logical.Backend, error) {
		return NewTokenStore(ctx, c, config)
	}
	c.credentialBackends = credentialBackends

	auditBackends := make(map[string]audit.Factory)
	for k, f := range conf.AuditBackends {
		auditBackends[k] = f
	}
	c.auditBackends = auditBackends

	return c, nil
}

// Shutdown is invoked when the Vault instance is about to be terminated. It
// should not be accessible as part of an API call as it will cause an availability
// problem. It is only used to gracefully quit in the case of HA so that failover
// happens as quickly as possible.
func (c *Core) Shutdown() error {
	c.stateLock.RLock()
	// Tell any requests that know about this to stop
	if c.activeContextCancelFunc != nil {
		c.activeContextCancelFunc()
	}
	c.stateLock.RUnlock()

	// Seal the Vault, causes a leader stepdown
	retChan := make(chan error)
	go func() {
		c.stateLock.Lock()
		defer c.stateLock.Unlock()
		retChan <- c.sealInternal()
	}()

	return <-retChan
}

// CORSConfig returns the current CORS configuration
func (c *Core) CORSConfig() *CORSConfig {
	return c.corsConfig
}

func (c *Core) GetContext() (context.Context, context.CancelFunc) {
	c.stateLock.RLock()
	defer c.stateLock.RUnlock()

	return context.WithCancel(c.activeContext)
}

// LookupToken returns the properties of the token from the token store. This
// is particularly useful to fetch the accessor of the client token and get it
// populated in the logical request along with the client token. The accessor
// of the client token can get audit logged.
func (c *Core) LookupToken(token string) (*TokenEntry, error) {
	if token == "" {
		return nil, fmt.Errorf("missing client token")
	}

	c.stateLock.RLock()
	defer c.stateLock.RUnlock()
	if c.sealed {
		return nil, consts.ErrSealed
	}
	if c.standby {
		return nil, consts.ErrStandby
	}

	// Many tests don't have a token store running
	if c.tokenStore == nil {
		return nil, nil
	}

	return c.tokenStore.Lookup(c.activeContext, token)
}

// fetchEntityAndDerivedPolicies returns the entity object for the given entity
// ID. If the entity is merged into a different entity object, the entity into
// which the given entity ID is merged into will be returned. This function
// also returns the cumulative list of policies that the entity is entitled to.
// This list includes the policies from the entity itself and from all the
// groups in which the given entity ID is a member of.
func (c *Core) fetchEntityAndDerivedPolicies(entityID string) (*identity.Entity, []string, error) {
	if entityID == "" {
		return nil, nil, nil
	}

	//c.logger.Debug("core: entity set on the token", "entity_id", te.EntityID)

	// Fetch the entity
	entity, err := c.identityStore.MemDBEntityByID(entityID, false)
	if err != nil {
		c.logger.Error("core: failed to lookup entity using its ID", "error", err)
		return nil, nil, err
	}

	if entity == nil {
		// If there was no corresponding entity object found, it is
		// possible that the entity got merged into another entity. Try
		// finding entity based on the merged entity index.
		entity, err = c.identityStore.MemDBEntityByMergedEntityID(entityID, false)
		if err != nil {
			c.logger.Error("core: failed to lookup entity in merged entity ID index", "error", err)
			return nil, nil, err
		}
	}

	var policies []string
	if entity != nil {
		//c.logger.Debug("core: entity successfully fetched; adding entity policies to token's policies to create ACL")

		// Attach the policies on the entity
		policies = append(policies, entity.Policies...)

		groupPolicies, err := c.identityStore.groupPoliciesByEntityID(entity.ID)
		if err != nil {
			c.logger.Error("core: failed to fetch group policies", "error", err)
			return nil, nil, err
		}

		// Attach the policies from all the groups
		policies = append(policies, groupPolicies...)
	}

	return entity, policies, err
}

func (c *Core) fetchACLTokenEntryAndEntity(clientToken string) (*ACL, *TokenEntry, *identity.Entity, error) {
	defer metrics.MeasureSince([]string{"core", "fetch_acl_and_token"}, time.Now())

	// Ensure there is a client token
	if clientToken == "" {
		return nil, nil, nil, fmt.Errorf("missing client token")
	}

	if c.tokenStore == nil {
		c.logger.Error("core: token store is unavailable")
		return nil, nil, nil, ErrInternalError
	}

	// Resolve the token policy
	te, err := c.tokenStore.Lookup(c.activeContext, clientToken)
	if err != nil {
		c.logger.Error("core: failed to lookup token", "error", err)
		return nil, nil, nil, ErrInternalError
	}

	// Ensure the token is valid
	if te == nil {
		return nil, nil, nil, logical.ErrPermissionDenied
	}

	tokenPolicies := te.Policies

	entity, derivedPolicies, err := c.fetchEntityAndDerivedPolicies(te.EntityID)
	if err != nil {
		return nil, nil, nil, ErrInternalError
	}

	tokenPolicies = append(tokenPolicies, derivedPolicies...)

	// Construct the corresponding ACL object
	acl, err := c.policyStore.ACL(c.activeContext, tokenPolicies...)
	if err != nil {
		c.logger.Error("core: failed to construct ACL", "error", err)
		return nil, nil, nil, ErrInternalError
	}

	return acl, te, entity, nil
}

func (c *Core) checkToken(ctx context.Context, req *logical.Request, unauth bool) (*logical.Auth, *TokenEntry, error) {
	defer metrics.MeasureSince([]string{"core", "check_token"}, time.Now())

	var acl *ACL
	var te *TokenEntry
	var entity *identity.Entity
	var err error

	// Even if unauth, if a token is provided, there's little reason not to
	// gather as much info as possible for the audit log and to e.g. control
	// trace mode for EGPs.
	if !unauth || (unauth && req.ClientToken != "") {
		acl, te, entity, err = c.fetchACLTokenEntryAndEntity(req.ClientToken)
		// In the unauth case we don't want to fail the command, since it's
		// unauth, we just have no information to attach to the request, so
		// ignore errors...this was best-effort anyways
		if err != nil && !unauth {
			return nil, te, err
		}
	}

	// Check if this is a root protected path
	rootPath := c.router.RootPath(req.Path)

	if rootPath && unauth {
		return nil, nil, errors.New("cannot access root path in unauthenticated request")
	}

	// When we receive a write of either type, rather than require clients to
	// PUT/POST and trust the operation, we ask the backend to give us the real
	// skinny -- if the backend implements an existence check, it can tell us
	// whether a particular resource exists. Then we can mark it as an update
	// or creation as appropriate.
	if req.Operation == logical.CreateOperation || req.Operation == logical.UpdateOperation {
		checkExists, resourceExists, err := c.router.RouteExistenceCheck(ctx, req)
		switch err {
		case logical.ErrUnsupportedPath:
			// fail later via bad path to avoid confusing items in the log
			checkExists = false
		case nil:
			// Continue on
		default:
			c.logger.Error("core: failed to run existence check", "error", err)
			if _, ok := err.(errutil.UserError); ok {
				return nil, nil, err
			} else {
				return nil, nil, ErrInternalError
			}
		}

		switch {
		case checkExists == false:
			// No existence check, so always treate it as an update operation, which is how it is pre 0.5
			req.Operation = logical.UpdateOperation
		case resourceExists == true:
			// It exists, so force an update operation
			req.Operation = logical.UpdateOperation
		case resourceExists == false:
			// It doesn't exist, force a create operation
			req.Operation = logical.CreateOperation
		default:
			panic("unreachable code")
		}
	}
	// Create the auth response
	auth := &logical.Auth{
		ClientToken: req.ClientToken,
		Accessor:    req.ClientTokenAccessor,
	}

	if te != nil {
		auth.Policies = te.Policies
		auth.Metadata = te.Meta
		auth.DisplayName = te.DisplayName
		auth.EntityID = te.EntityID
		// Store the entity ID in the request object
		req.EntityID = te.EntityID
	}

	// Check the standard non-root ACLs. Return the token entry if it's not
	// allowed so we can decrement the use count.
	authResults := c.performPolicyChecks(ctx, acl, te, req, entity, &PolicyCheckOpts{
		Unauth:            unauth,
		RootPrivsRequired: rootPath,
	})
	if authResults.Error.ErrorOrNil() != nil {
		return auth, te, authResults.Error
	}
	if !authResults.Allowed {
		// Return auth for audit logging even if not allowed
		return auth, te, logical.ErrPermissionDenied
	}

	return auth, te, nil
}

// Sealed checks if the Vault is current sealed
func (c *Core) Sealed() (bool, error) {
	c.stateLock.RLock()
	defer c.stateLock.RUnlock()
	return c.sealed, nil
}

// Standby checks if the Vault is in standby mode
func (c *Core) Standby() (bool, error) {
	c.stateLock.RLock()
	defer c.stateLock.RUnlock()
	return c.standby, nil
}

// Leader is used to get the current active leader
func (c *Core) Leader() (isLeader bool, leaderAddr, clusterAddr string, err error) {
	c.stateLock.RLock()
	defer c.stateLock.RUnlock()

	// Check if sealed
	if c.sealed {
		return false, "", "", consts.ErrSealed
	}

	// Check if HA enabled
	if c.ha == nil {
		return false, "", "", ErrHANotEnabled
	}

	// Check if we are the leader
	if !c.standby {
		return true, c.redirectAddr, c.clusterAddr, nil
	}

	// Initialize a lock
	lock, err := c.ha.LockWith(coreLockPath, "read")
	if err != nil {
		return false, "", "", err
	}

	// Read the value
	held, leaderUUID, err := lock.Value()
	if err != nil {
		return false, "", "", err
	}
	if !held {
		return false, "", "", nil
	}

	c.clusterLeaderParamsLock.RLock()
	localLeaderUUID := c.clusterLeaderUUID
	localRedirAddr := c.clusterLeaderRedirectAddr
	localClusterAddr := c.clusterLeaderClusterAddr
	c.clusterLeaderParamsLock.RUnlock()

	// If the leader hasn't changed, return the cached value; nothing changes
	// mid-leadership, and the barrier caches anyways
	if leaderUUID == localLeaderUUID && localRedirAddr != "" {
		return false, localRedirAddr, localClusterAddr, nil
	}

	c.logger.Trace("core: found new active node information, refreshing")

	c.clusterLeaderParamsLock.Lock()
	defer c.clusterLeaderParamsLock.Unlock()

	// Validate base conditions again
	if leaderUUID == c.clusterLeaderUUID && c.clusterLeaderRedirectAddr != "" {
		return false, localRedirAddr, localClusterAddr, nil
	}

	key := coreLeaderPrefix + leaderUUID
	// Use background because postUnseal isn't run on standby
	entry, err := c.barrier.Get(context.Background(), key)
	if err != nil {
		return false, "", "", err
	}
	if entry == nil {
		return false, "", "", nil
	}

	var oldAdv bool

	var adv activeAdvertisement
	err = jsonutil.DecodeJSON(entry.Value, &adv)
	if err != nil {
		// Fall back to pre-struct handling
		adv.RedirectAddr = string(entry.Value)
		c.logger.Trace("core: parsed redirect addr for new active node", "redirect_addr", adv.RedirectAddr)
		oldAdv = true
	}

	if !oldAdv {
		c.logger.Trace("core: parsing information for new active node", "active_cluster_addr", adv.ClusterAddr, "active_redirect_addr", adv.RedirectAddr)

		// Ensure we are using current values
		err = c.loadLocalClusterTLS(adv)
		if err != nil {
			return false, "", "", err
		}

		// This will ensure that we both have a connection at the ready and that
		// the address is the current known value
		// Since this is standby, we don't use the active context. Later we may
		// use a process-scoped context
		err = c.refreshRequestForwardingConnection(context.Background(), adv.ClusterAddr)
		if err != nil {
			return false, "", "", err
		}
	}

	// Don't set these until everything has been parsed successfully or we'll
	// never try again
	c.clusterLeaderRedirectAddr = adv.RedirectAddr
	c.clusterLeaderClusterAddr = adv.ClusterAddr
	c.clusterLeaderUUID = leaderUUID

	return false, adv.RedirectAddr, adv.ClusterAddr, nil
}

// SecretProgress returns the number of keys provided so far
func (c *Core) SecretProgress() (int, string) {
	c.stateLock.RLock()
	defer c.stateLock.RUnlock()
	switch c.unlockInfo {
	case nil:
		return 0, ""
	default:
		return len(c.unlockInfo.Parts), c.unlockInfo.Nonce
	}
}

// ResetUnsealProcess removes the current unlock parts from memory, to reset
// the unsealing process
func (c *Core) ResetUnsealProcess() {
	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	if !c.sealed {
		return
	}
	c.unlockInfo = nil
}

// Unseal is used to provide one of the key parts to unseal the Vault.
//
// They key given as a parameter will automatically be zerod after
// this method is done with it. If you want to keep the key around, a copy
// should be made.
func (c *Core) Unseal(key []byte) (bool, error) {
	defer metrics.MeasureSince([]string{"core", "unseal"}, time.Now())

	c.stateLock.Lock()
	defer c.stateLock.Unlock()

	ctx := context.Background()

	// Explicitly check for init status. This also checks if the seal
	// configuration is valid (i.e. non-nil).
	init, err := c.Initialized(ctx)
	if err != nil {
		return false, err
	}
	if !init {
		return false, ErrNotInit
	}

	// Verify the key length
	min, max := c.barrier.KeyLength()
	max += shamir.ShareOverhead
	if len(key) < min {
		return false, &ErrInvalidKey{fmt.Sprintf("key is shorter than minimum %d bytes", min)}
	}
	if len(key) > max {
		return false, &ErrInvalidKey{fmt.Sprintf("key is longer than maximum %d bytes", max)}
	}

	// Get the barrier seal configuration
	config, err := c.seal.BarrierConfig(ctx)
	if err != nil {
		return false, err
	}

	// Check if already unsealed
	if !c.sealed {
		return true, nil
	}

	masterKey, err := c.unsealPart(ctx, config, key, false)
	if err != nil {
		return false, err
	}
	if masterKey != nil {
		return c.unsealInternal(ctx, masterKey)
	}

	return false, nil
}

// UnsealWithRecoveryKeys is used to provide one of the recovery key shares to
// unseal the Vault.
func (c *Core) UnsealWithRecoveryKeys(ctx context.Context, key []byte) (bool, error) {
	defer metrics.MeasureSince([]string{"core", "unseal_with_recovery_keys"}, time.Now())

	c.stateLock.Lock()
	defer c.stateLock.Unlock()

	// Explicitly check for init status
	init, err := c.Initialized(ctx)
	if err != nil {
		return false, err
	}
	if !init {
		return false, ErrNotInit
	}

	var config *SealConfig
	// If recovery keys are supported then use recovery seal config to unseal
	if c.seal.RecoveryKeySupported() {
		config, err = c.seal.RecoveryConfig(ctx)
		if err != nil {
			return false, err
		}
	}

	// Check if already unsealed
	if !c.sealed {
		return true, nil
	}

	masterKey, err := c.unsealPart(ctx, config, key, true)
	if err != nil {
		return false, err
	}
	if masterKey != nil {
		return c.unsealInternal(ctx, masterKey)
	}

	return false, nil
}

// unsealPart takes in a key share, and returns the master key if the threshold
// is met. If recovery keys are supported, recovery key shares may be provided.
func (c *Core) unsealPart(ctx context.Context, config *SealConfig, key []byte, useRecoveryKeys bool) ([]byte, error) {
	// Check if we already have this piece
	if c.unlockInfo != nil {
		for _, existing := range c.unlockInfo.Parts {
			if subtle.ConstantTimeCompare(existing, key) == 1 {
				return nil, nil
			}
		}
	} else {
		uuid, err := uuid.GenerateUUID()
		if err != nil {
			return nil, err
		}
		c.unlockInfo = &unlockInformation{
			Nonce: uuid,
		}
	}

	// Store this key
	c.unlockInfo.Parts = append(c.unlockInfo.Parts, key)

	// Check if we don't have enough keys to unlock, proceed through the rest of
	// the call only if we have met the threshold
	if len(c.unlockInfo.Parts) < config.SecretThreshold {
		if c.logger.IsDebug() {
			c.logger.Debug("core: cannot unseal, not enough keys", "keys", len(c.unlockInfo.Parts), "threshold", config.SecretThreshold, "nonce", c.unlockInfo.Nonce)
		}
		return nil, nil
	}

	// Best-effort memzero of unlock parts once we're done with them
	defer func() {
		for i := range c.unlockInfo.Parts {
			memzero(c.unlockInfo.Parts[i])
		}
		c.unlockInfo = nil
	}()

	// Recover the split key. recoveredKey is the shamir combined
	// key, or the single provided key if the threshold is 1.
	var recoveredKey []byte
	var err error
	if config.SecretThreshold == 1 {
		recoveredKey = make([]byte, len(c.unlockInfo.Parts[0]))
		copy(recoveredKey, c.unlockInfo.Parts[0])
	} else {
		recoveredKey, err = shamir.Combine(c.unlockInfo.Parts)
		if err != nil {
			return nil, fmt.Errorf("failed to compute master key: %v", err)
		}
	}

	if c.seal.RecoveryKeySupported() && useRecoveryKeys {
		// Verify recovery key
		if err := c.seal.VerifyRecoveryKey(ctx, recoveredKey); err != nil {
			return nil, err
		}

		// Get stored keys and shamir combine into single master key. Unsealing with
		// recovery keys currently does not support: 1) mixed stored and non-stored
		// keys setup, nor 2) seals that support recovery keys but not stored keys.
		// If insuffiencient shares are provided, shamir.Combine will error, and if
		// no stored keys are found it will return masterKey as nil.
		var masterKey []byte
		if c.seal.StoredKeysSupported() {
			masterKeyShares, err := c.seal.GetStoredKeys(ctx)
			if err != nil {
				return nil, fmt.Errorf("unable to retrieve stored keys: %v", err)
			}

			if len(masterKeyShares) == 1 {
				return masterKeyShares[0], nil
			}

			masterKey, err = shamir.Combine(masterKeyShares)
			if err != nil {
				return nil, fmt.Errorf("failed to compute master key: %v", err)
			}
		}
		return masterKey, nil
	}

	// If this is not a recovery key-supported seal, then the recovered key is
	// the master key to be returned.
	return recoveredKey, nil
}

// unsealInternal takes in the master key and attempts to unseal the barrier.
// N.B.: This must be called with the state write lock held.
func (c *Core) unsealInternal(ctx context.Context, masterKey []byte) (bool, error) {
	defer memzero(masterKey)

	// Attempt to unlock
	if err := c.barrier.Unseal(ctx, masterKey); err != nil {
		return false, err
	}
	if c.logger.IsInfo() {
		c.logger.Info("core: vault is unsealed")
	}

	// Do post-unseal setup if HA is not enabled
	if c.ha == nil {
		// We still need to set up cluster info even if it's not part of a
		// cluster right now. This also populates the cached cluster object.
		if err := c.setupCluster(ctx); err != nil {
			c.logger.Error("core: cluster setup failed", "error", err)
			c.barrier.Seal()
			c.logger.Warn("core: vault is sealed")
			return false, err
		}

		if err := c.postUnseal(); err != nil {
			c.logger.Error("core: post-unseal setup failed", "error", err)
			c.barrier.Seal()
			c.logger.Warn("core: vault is sealed")
			return false, err
		}

		c.standby = false
	} else {
		// Go to standby mode, wait until we are active to unseal
		c.standbyDoneCh = make(chan struct{})
		c.standbyStopCh = make(chan struct{})
		c.manualStepDownCh = make(chan struct{})
		go c.runStandby(c.standbyDoneCh, c.standbyStopCh, c.manualStepDownCh)
	}

	// Success!
	c.sealed = false

	// Force a cache bust here, which will also run migration code
	if c.seal.RecoveryKeySupported() {
		c.seal.SetRecoveryConfig(ctx, nil)
	}

	if c.ha != nil {
		sd, ok := c.ha.(physical.ServiceDiscovery)
		if ok {
			if err := sd.NotifySealedStateChange(); err != nil {
				if c.logger.IsWarn() {
					c.logger.Warn("core: failed to notify unsealed status", "error", err)
				}
			}
		}
	}
	return true, nil
}

// SealWithRequest takes in a logical.Request, acquires the lock, and passes
// through to sealInternal
func (c *Core) SealWithRequest(req *logical.Request) error {
	defer metrics.MeasureSince([]string{"core", "seal-with-request"}, time.Now())

	c.stateLock.RLock()

	if c.sealed {
		c.stateLock.RUnlock()
		return nil
	}

	// This will unlock the read lock
	// We use background context since we may not be active
	return c.sealInitCommon(context.Background(), req)
}

// Seal takes in a token and creates a logical.Request, acquires the lock, and
// passes through to sealInternal
func (c *Core) Seal(token string) error {
	defer metrics.MeasureSince([]string{"core", "seal"}, time.Now())

	c.stateLock.RLock()

	if c.sealed {
		c.stateLock.RUnlock()
		return nil
	}

	req := &logical.Request{
		Operation:   logical.UpdateOperation,
		Path:        "sys/seal",
		ClientToken: token,
	}

	// This will unlock the read lock
	// We use background context since we may not be active
	return c.sealInitCommon(context.Background(), req)
}

// sealInitCommon is common logic for Seal and SealWithRequest and is used to
// re-seal the Vault. This requires the Vault to be unsealed again to perform
// any further operations. Note: this function will read-unlock the state lock.
func (c *Core) sealInitCommon(ctx context.Context, req *logical.Request) (retErr error) {
	defer metrics.MeasureSince([]string{"core", "seal-internal"}, time.Now())

	if req == nil {
		retErr = multierror.Append(retErr, errors.New("nil request to seal"))
		c.stateLock.RUnlock()
		return retErr
	}

	// Validate the token is a root token
	acl, te, entity, err := c.fetchACLTokenEntryAndEntity(req.ClientToken)
	if err != nil {
		// Since there is no token store in standby nodes, sealing cannot
		// be done. Ideally, the request has to be forwarded to leader node
		// for validation and the operation should be performed. But for now,
		// just returning with an error and recommending a vault restart, which
		// essentially does the same thing.
		if c.standby {
			c.logger.Error("core: vault cannot seal when in standby mode; please restart instead")
			retErr = multierror.Append(retErr, errors.New("vault cannot seal when in standby mode; please restart instead"))
			c.stateLock.RUnlock()
			return retErr
		}
		retErr = multierror.Append(retErr, err)
		c.stateLock.RUnlock()
		return retErr
	}

	// Audit-log the request before going any further
	auth := &logical.Auth{
		ClientToken: req.ClientToken,
		Policies:    te.Policies,
		Metadata:    te.Meta,
		DisplayName: te.DisplayName,
		EntityID:    te.EntityID,
	}

	logInput := &audit.LogInput{
		Auth:    auth,
		Request: req,
	}
	if err := c.auditBroker.LogRequest(ctx, logInput, c.auditedHeaders); err != nil {
		c.logger.Error("core: failed to audit request", "request_path", req.Path, "error", err)
		retErr = multierror.Append(retErr, errors.New("failed to audit request, cannot continue"))
		c.stateLock.RUnlock()
		return retErr
	}

	// Attempt to use the token (decrement num_uses)
	// On error bail out; if the token has been revoked, bail out too
	if te != nil {
		te, err = c.tokenStore.UseToken(ctx, te)
		if err != nil {
			c.logger.Error("core: failed to use token", "error", err)
			retErr = multierror.Append(retErr, ErrInternalError)
			c.stateLock.RUnlock()
			return retErr
		}
		if te == nil {
			// Token is no longer valid
			retErr = multierror.Append(retErr, logical.ErrPermissionDenied)
			c.stateLock.RUnlock()
			return retErr
		}
	}

	// Verify that this operation is allowed
	authResults := c.performPolicyChecks(ctx, acl, te, req, entity, &PolicyCheckOpts{
		RootPrivsRequired: true,
	})
	if authResults.Error.ErrorOrNil() != nil {
		retErr = multierror.Append(retErr, authResults.Error)
		c.stateLock.RUnlock()
		return retErr
	}
	if !authResults.Allowed {
		retErr = multierror.Append(retErr, logical.ErrPermissionDenied)
		c.stateLock.RUnlock()
		return retErr
	}

	if te != nil && te.NumUses == -1 {
		// Token needs to be revoked. We do this immediately here because
		// we won't have a token store after sealing.
		err = c.tokenStore.Revoke(c.activeContext, te.ID)
		if err != nil {
			c.logger.Error("core: token needed revocation before seal but failed to revoke", "error", err)
			retErr = multierror.Append(retErr, ErrInternalError)
		}
	}

	// Tell any requests that know about this to stop
	if c.activeContextCancelFunc != nil {
		c.activeContextCancelFunc()
	}

	// Unlock from the request handling
	c.stateLock.RUnlock()

	//Seal the Vault
	retChan := make(chan error)
	go func() {
		c.stateLock.Lock()
		defer c.stateLock.Unlock()
		retChan <- c.sealInternal()
	}()

	funcErr := <-retChan
	if funcErr != nil {
		retErr = multierror.Append(retErr, funcErr)
	}

	return retErr
}

// StepDown is used to step down from leadership
func (c *Core) StepDown(req *logical.Request) (retErr error) {
	defer metrics.MeasureSince([]string{"core", "step_down"}, time.Now())

	if req == nil {
		retErr = multierror.Append(retErr, errors.New("nil request to step-down"))
		return retErr
	}

	c.stateLock.RLock()
	defer c.stateLock.RUnlock()
	if c.sealed {
		return nil
	}
	if c.ha == nil || c.standby {
		return nil
	}

	ctx := c.activeContext

	acl, te, entity, err := c.fetchACLTokenEntryAndEntity(req.ClientToken)
	if err != nil {
		retErr = multierror.Append(retErr, err)
		return retErr
	}

	// Audit-log the request before going any further
	auth := &logical.Auth{
		ClientToken: req.ClientToken,
		Policies:    te.Policies,
		Metadata:    te.Meta,
		DisplayName: te.DisplayName,
		EntityID:    te.EntityID,
	}

	logInput := &audit.LogInput{
		Auth:    auth,
		Request: req,
	}
	if err := c.auditBroker.LogRequest(ctx, logInput, c.auditedHeaders); err != nil {
		c.logger.Error("core: failed to audit request", "request_path", req.Path, "error", err)
		retErr = multierror.Append(retErr, errors.New("failed to audit request, cannot continue"))
		return retErr
	}

	// Attempt to use the token (decrement num_uses)
	if te != nil {
		te, err = c.tokenStore.UseToken(ctx, te)
		if err != nil {
			c.logger.Error("core: failed to use token", "error", err)
			retErr = multierror.Append(retErr, ErrInternalError)
			return retErr
		}
		if te == nil {
			// Token has been revoked
			retErr = multierror.Append(retErr, logical.ErrPermissionDenied)
			return retErr
		}
	}

	// Verify that this operation is allowed
	authResults := c.performPolicyChecks(ctx, acl, te, req, entity, &PolicyCheckOpts{
		RootPrivsRequired: true,
	})
	if authResults.Error.ErrorOrNil() != nil {
		retErr = multierror.Append(retErr, authResults.Error)
		return retErr
	}
	if !authResults.Allowed {
		retErr = multierror.Append(retErr, logical.ErrPermissionDenied)
		return retErr
	}

	if te != nil && te.NumUses == -1 {
		// Token needs to be revoked. We do this immediately here because
		// we won't have a token store after sealing.
		err = c.tokenStore.Revoke(c.activeContext, te.ID)
		if err != nil {
			c.logger.Error("core: token needed revocation before step-down but failed to revoke", "error", err)
			retErr = multierror.Append(retErr, ErrInternalError)
		}
	}

	select {
	case c.manualStepDownCh <- struct{}{}:
	default:
		c.logger.Warn("core: manual step-down operation already queued")
	}

	return retErr
}

// sealInternal is an internal method used to seal the vault.  It does not do
// any authorization checking. The stateLock must be held prior to calling.
func (c *Core) sealInternal() error {
	if c.sealed {
		return nil
	}

	// Enable that we are sealed to prevent further transactions
	c.sealed = true

	c.logger.Debug("core: marked as sealed")

	// Clear forwarding clients
	c.requestForwardingConnectionLock.Lock()
	c.clearForwardingClients()
	c.requestForwardingConnectionLock.Unlock()

	// Do pre-seal teardown if HA is not enabled
	if c.ha == nil {
		// Even in a non-HA context we key off of this for some things
		c.standby = true
		if err := c.preSeal(); err != nil {
			c.logger.Error("core: pre-seal teardown failed", "error", err)
			return fmt.Errorf("internal error")
		}
	} else {
		// Signal the standby goroutine to shutdown, wait for completion
		close(c.standbyStopCh)

		// Release the lock while we wait to avoid deadlocking
		c.stateLock.Unlock()
		<-c.standbyDoneCh
		c.stateLock.Lock()
	}

	c.logger.Debug("core: sealing barrier")
	if err := c.barrier.Seal(); err != nil {
		c.logger.Error("core: error sealing barrier", "error", err)
		return err
	}

	if c.ha != nil {
		sd, ok := c.ha.(physical.ServiceDiscovery)
		if ok {
			if err := sd.NotifySealedStateChange(); err != nil {
				if c.logger.IsWarn() {
					c.logger.Warn("core: failed to notify sealed status", "error", err)
				}
			}
		}
	}

	c.logger.Info("core: vault is sealed")

	return nil
}

// postUnseal is invoked after the barrier is unsealed, but before
// allowing any user operations. This allows us to setup any state that
// requires the Vault to be unsealed such as mount tables, logical backends,
// credential stores, etc.
func (c *Core) postUnseal() (retErr error) {
	defer metrics.MeasureSince([]string{"core", "post_unseal"}, time.Now())

	// Create a new request context
	c.activeContext, c.activeContextCancelFunc = context.WithCancel(context.Background())

	defer func() {
		if retErr != nil {
			c.activeContextCancelFunc()
			c.preSeal()
		}
	}()
	c.logger.Info("core: post-unseal setup starting")

	// Clear forwarding clients; we're active
	c.requestForwardingConnectionLock.Lock()
	c.clearForwardingClients()
	c.requestForwardingConnectionLock.Unlock()

	c.physicalCache.Purge(c.activeContext)
	if !c.cachingDisabled {
		c.physicalCache.SetEnabled(true)
	}

	switch c.sealUnwrapper.(type) {
	case *sealUnwrapper:
		c.sealUnwrapper.(*sealUnwrapper).runUnwraps()
	case *transactionalSealUnwrapper:
		c.sealUnwrapper.(*transactionalSealUnwrapper).runUnwraps()
	}

	// Purge these for safety in case of a rekey
	c.seal.SetBarrierConfig(c.activeContext, nil)
	if c.seal.RecoveryKeySupported() {
		c.seal.SetRecoveryConfig(c.activeContext, nil)
	}

	if err := enterprisePostUnseal(c); err != nil {
		return err
	}
	if err := c.ensureWrappingKey(c.activeContext); err != nil {
		return err
	}
	if err := c.setupPluginCatalog(); err != nil {
		return err
	}
	if err := c.loadMounts(c.activeContext); err != nil {
		return err
	}
	if err := c.setupMounts(c.activeContext); err != nil {
		return err
	}
	if err := c.setupPolicyStore(c.activeContext); err != nil {
		return err
	}
	if err := c.loadCORSConfig(c.activeContext); err != nil {
		return err
	}
	if err := c.loadCredentials(c.activeContext); err != nil {
		return err
	}
	if err := c.setupCredentials(c.activeContext); err != nil {
		return err
	}
	if err := c.startRollback(); err != nil {
		return err
	}
	if err := c.setupExpiration(); err != nil {
		return err
	}
	if err := c.loadAudits(c.activeContext); err != nil {
		return err
	}
	if err := c.setupAudits(c.activeContext); err != nil {
		return err
	}
	if err := c.loadIdentityStoreArtifacts(c.activeContext); err != nil {
		return err
	}
	if err := c.setupAuditedHeadersConfig(c.activeContext); err != nil {
		return err
	}

	if c.ha != nil {
		if err := c.startClusterListener(c.activeContext); err != nil {
			return err
		}
	}
	c.metricsCh = make(chan struct{})
	go c.emitMetrics(c.metricsCh)
	c.logger.Info("core: post-unseal setup complete")
	return nil
}

// preSeal is invoked before the barrier is sealed, allowing
// for any state teardown required.
func (c *Core) preSeal() error {
	defer metrics.MeasureSince([]string{"core", "pre_seal"}, time.Now())
	c.logger.Info("core: pre-seal teardown starting")

	// Clear any rekey progress
	c.barrierRekeyConfig = nil
	c.barrierRekeyProgress = nil
	c.recoveryRekeyConfig = nil
	c.recoveryRekeyProgress = nil

	if c.metricsCh != nil {
		close(c.metricsCh)
		c.metricsCh = nil
	}
	var result error

	c.stopClusterListener()

	if err := c.teardownAudits(); err != nil {
		result = multierror.Append(result, errwrap.Wrapf("error tearing down audits: {{err}}", err))
	}
	if err := c.stopExpiration(); err != nil {
		result = multierror.Append(result, errwrap.Wrapf("error stopping expiration: {{err}}", err))
	}
	if err := c.teardownCredentials(c.activeContext); err != nil {
		result = multierror.Append(result, errwrap.Wrapf("error tearing down credentials: {{err}}", err))
	}
	if err := c.teardownPolicyStore(); err != nil {
		result = multierror.Append(result, errwrap.Wrapf("error tearing down policy store: {{err}}", err))
	}
	if err := c.stopRollback(); err != nil {
		result = multierror.Append(result, errwrap.Wrapf("error stopping rollback: {{err}}", err))
	}
	if err := c.unloadMounts(c.activeContext); err != nil {
		result = multierror.Append(result, errwrap.Wrapf("error unloading mounts: {{err}}", err))
	}
	if err := enterprisePreSeal(c); err != nil {
		result = multierror.Append(result, err)
	}

	switch c.sealUnwrapper.(type) {
	case *sealUnwrapper:
		c.sealUnwrapper.(*sealUnwrapper).stopUnwraps()
	case *transactionalSealUnwrapper:
		c.sealUnwrapper.(*transactionalSealUnwrapper).stopUnwraps()
	}

	// Purge the cache
	c.physicalCache.SetEnabled(false)
	c.physicalCache.Purge(c.activeContext)

	c.logger.Info("core: pre-seal teardown complete")
	return result
}

func enterprisePostUnsealImpl(c *Core) error {
	return nil
}

func enterprisePreSealImpl(c *Core) error {
	return nil
}

func startReplicationImpl(c *Core) error {
	return nil
}

func stopReplicationImpl(c *Core) error {
	return nil
}

// runStandby is a long running routine that is used when an HA backend
// is enabled. It waits until we are leader and switches this Vault to
// active.
func (c *Core) runStandby(doneCh, stopCh, manualStepDownCh chan struct{}) {
	defer close(doneCh)
	defer close(manualStepDownCh)
	c.logger.Info("core: entering standby mode")

	// Monitor for key rotation
	keyRotateDone := make(chan struct{})
	keyRotateStop := make(chan struct{})
	go c.periodicCheckKeyUpgrade(context.Background(), keyRotateDone, keyRotateStop)
	// Monitor for new leadership
	checkLeaderDone := make(chan struct{})
	checkLeaderStop := make(chan struct{})
	go c.periodicLeaderRefresh(checkLeaderDone, checkLeaderStop)
	defer func() {
		close(keyRotateStop)
		<-keyRotateDone
		close(checkLeaderStop)
		<-checkLeaderDone
	}()

	for {
		// Check for a shutdown
		select {
		case <-stopCh:
			return
		default:
		}

		// Create a lock
		uuid, err := uuid.GenerateUUID()
		if err != nil {
			c.logger.Error("core: failed to generate uuid", "error", err)
			return
		}
		lock, err := c.ha.LockWith(coreLockPath, uuid)
		if err != nil {
			c.logger.Error("core: failed to create lock", "error", err)
			return
		}

		// Attempt the acquisition
		leaderLostCh := c.acquireLock(lock, stopCh)

		// Bail if we are being shutdown
		if leaderLostCh == nil {
			return
		}
		c.logger.Info("core: acquired lock, enabling active operation")

		// This is used later to log a metrics event; this can be helpful to
		// detect flapping
		activeTime := time.Now()

		// Grab the lock as we need it for cluster setup, which needs to happen
		// before advertising;
		c.stateLock.Lock()

		// We haven't run postUnseal yet so we have nothing meaningful to use here
		ctx := context.Background()

		// This block is used to wipe barrier/seal state and verify that
		// everything is sane. If we have no sanity in the barrier, we actually
		// seal, as there's little we can do.
		{
			c.seal.SetBarrierConfig(ctx, nil)
			if c.seal.RecoveryKeySupported() {
				c.seal.SetRecoveryConfig(ctx, nil)
			}

			if err := c.performKeyUpgrades(ctx); err != nil {
				// We call this in a goroutine so that we can give up the
				// statelock and have this shut us down; sealInternal has a
				// workflow where it watches for the stopCh to close so we want
				// to return from here
				go c.Shutdown()
				c.logger.Error("core: error performing key upgrades", "error", err)
				c.stateLock.Unlock()
				lock.Unlock()
				metrics.MeasureSince([]string{"core", "leadership_setup_failed"}, activeTime)
				return
			}
		}

		// Clear previous local cluster cert info so we generate new. Since the
		// UUID will have changed, standbys will know to look for new info
		c.localClusterParsedCert.Store((*x509.Certificate)(nil))
		c.localClusterCert.Store(([]byte)(nil))
		c.localClusterPrivateKey.Store((*ecdsa.PrivateKey)(nil))

		if err := c.setupCluster(ctx); err != nil {
			c.stateLock.Unlock()
			c.logger.Error("core: cluster setup failed", "error", err)
			lock.Unlock()
			metrics.MeasureSince([]string{"core", "leadership_setup_failed"}, activeTime)
			continue
		}

		// Advertise as leader
		if err := c.advertiseLeader(ctx, uuid, leaderLostCh); err != nil {
			c.stateLock.Unlock()
			c.logger.Error("core: leader advertisement setup failed", "error", err)
			lock.Unlock()
			metrics.MeasureSince([]string{"core", "leadership_setup_failed"}, activeTime)
			continue
		}

		// Attempt the post-unseal process
		err = c.postUnseal()
		if err == nil {
			c.standby = false
		}
		c.stateLock.Unlock()

		// Handle a failure to unseal
		if err != nil {
			c.logger.Error("core: post-unseal setup failed", "error", err)
			lock.Unlock()
			metrics.MeasureSince([]string{"core", "leadership_setup_failed"}, activeTime)
			continue
		}

		// Monitor a loss of leadership
		var manualStepDown bool
		select {
		case <-leaderLostCh:
			c.logger.Warn("core: leadership lost, stopping active operation")
		case <-stopCh:
			c.logger.Warn("core: stopping active operation")
		case <-manualStepDownCh:
			c.logger.Warn("core: stepping down from active operation to standby")
			manualStepDown = true
		}

		metrics.MeasureSince([]string{"core", "leadership_lost"}, activeTime)

		// Clear ourself as leader
		if err := c.clearLeader(uuid); err != nil {
			c.logger.Error("core: clearing leader advertisement failed", "error", err)
		}

		// Tell any requests that know about this to stop
		if c.activeContextCancelFunc != nil {
			c.activeContextCancelFunc()
		}

		// Attempt the pre-seal process
		c.stateLock.Lock()
		c.standby = true
		preSealErr := c.preSeal()
		c.stateLock.Unlock()

		// Give up leadership
		lock.Unlock()

		// Check for a failure to prepare to seal
		if preSealErr != nil {
			c.logger.Error("core: pre-seal teardown failed", "error", err)
		}

		// If we've merely stepped down, we could instantly grab the lock
		// again. Give the other nodes a chance.
		if manualStepDown {
			time.Sleep(manualStepDownSleepPeriod)
		}
	}
}

// This checks the leader periodically to ensure that we switch RPC to a new
// leader pretty quickly. There is logic in Leader() already to not make this
// onerous and avoid more traffic than needed, so we just call that and ignore
// the result.
func (c *Core) periodicLeaderRefresh(doneCh, stopCh chan struct{}) {
	defer close(doneCh)
	for {
		select {
		case <-time.After(leaderCheckInterval):
			c.Leader()
		case <-stopCh:
			return
		}
	}
}

// periodicCheckKeyUpgrade is used to watch for key rotation events as a standby
func (c *Core) periodicCheckKeyUpgrade(ctx context.Context, doneCh, stopCh chan struct{}) {
	defer close(doneCh)
	for {
		select {
		case <-time.After(keyRotateCheckInterval):
			// Only check if we are a standby
			c.stateLock.RLock()
			standby := c.standby
			c.stateLock.RUnlock()
			if !standby {
				continue
			}

			// Check for a poison pill. If we can read it, it means we have stale
			// keys (e.g. from replication being activated) and we need to seal to
			// be unsealed again.
			entry, _ := c.barrier.Get(ctx, poisonPillPath)
			if entry != nil && len(entry.Value) > 0 {
				c.logger.Warn("core: encryption keys have changed out from underneath us (possibly due to replication enabling), must be unsealed again")
				go c.Shutdown()
				continue
			}

			if err := c.checkKeyUpgrades(ctx); err != nil {
				c.logger.Error("core: key rotation periodic upgrade check failed", "error", err)
			}
		case <-stopCh:
			return
		}
	}
}

// checkKeyUpgrades is used to check if there have been any key rotations
// and if there is a chain of upgrades available
func (c *Core) checkKeyUpgrades(ctx context.Context) error {
	for {
		// Check for an upgrade
		didUpgrade, newTerm, err := c.barrier.CheckUpgrade(ctx)
		if err != nil {
			return err
		}

		// Nothing to do if no upgrade
		if !didUpgrade {
			break
		}
		if c.logger.IsInfo() {
			c.logger.Info("core: upgraded to new key term", "term", newTerm)
		}
	}
	return nil
}

// scheduleUpgradeCleanup is used to ensure that all the upgrade paths
// are cleaned up in a timely manner if a leader failover takes place
func (c *Core) scheduleUpgradeCleanup(ctx context.Context) error {
	// List the upgrades
	upgrades, err := c.barrier.List(ctx, keyringUpgradePrefix)
	if err != nil {
		return fmt.Errorf("failed to list upgrades: %v", err)
	}

	// Nothing to do if no upgrades
	if len(upgrades) == 0 {
		return nil
	}

	// Schedule cleanup for all of them
	time.AfterFunc(keyRotateGracePeriod, func() {
		sealed, err := c.barrier.Sealed()
		if err != nil {
			c.logger.Warn("core: failed to check barrier status at upgrade cleanup time")
			return
		}
		if sealed {
			c.logger.Warn("core: barrier sealed at upgrade cleanup time")
			return
		}
		for _, upgrade := range upgrades {
			path := fmt.Sprintf("%s%s", keyringUpgradePrefix, upgrade)
			if err := c.barrier.Delete(ctx, path); err != nil {
				c.logger.Error("core: failed to cleanup upgrade", "path", path, "error", err)
			}
		}
	})
	return nil
}

func (c *Core) performKeyUpgrades(ctx context.Context) error {
	if err := c.checkKeyUpgrades(ctx); err != nil {
		return errwrap.Wrapf("error checking for key upgrades: {{err}}", err)
	}

	if err := c.barrier.ReloadMasterKey(ctx); err != nil {
		return errwrap.Wrapf("error reloading master key: {{err}}", err)
	}

	if err := c.barrier.ReloadKeyring(ctx); err != nil {
		return errwrap.Wrapf("error reloading keyring: {{err}}", err)
	}

	if err := c.scheduleUpgradeCleanup(ctx); err != nil {
		return errwrap.Wrapf("error scheduling upgrade cleanup: {{err}}", err)
	}

	return nil
}

// acquireLock blocks until the lock is acquired, returning the leaderLostCh
func (c *Core) acquireLock(lock physical.Lock, stopCh <-chan struct{}) <-chan struct{} {
	for {
		// Attempt lock acquisition
		leaderLostCh, err := lock.Lock(stopCh)
		if err == nil {
			return leaderLostCh
		}

		// Retry the acquisition
		c.logger.Error("core: failed to acquire lock", "error", err)
		select {
		case <-time.After(lockRetryInterval):
		case <-stopCh:
			return nil
		}
	}
}

// advertiseLeader is used to advertise the current node as leader
func (c *Core) advertiseLeader(ctx context.Context, uuid string, leaderLostCh <-chan struct{}) error {
	go c.cleanLeaderPrefix(ctx, uuid, leaderLostCh)

	var key *ecdsa.PrivateKey
	switch c.localClusterPrivateKey.Load().(type) {
	case *ecdsa.PrivateKey:
		key = c.localClusterPrivateKey.Load().(*ecdsa.PrivateKey)
	default:
		c.logger.Error("core: unknown cluster private key type", "key_type", fmt.Sprintf("%T", c.localClusterPrivateKey.Load()))
		return fmt.Errorf("unknown cluster private key type %T", c.localClusterPrivateKey.Load())
	}

	keyParams := &clusterKeyParams{
		Type: corePrivateKeyTypeP521,
		X:    key.X,
		Y:    key.Y,
		D:    key.D,
	}

	locCert := c.localClusterCert.Load().([]byte)
	localCert := make([]byte, len(locCert))
	copy(localCert, locCert)
	adv := &activeAdvertisement{
		RedirectAddr:     c.redirectAddr,
		ClusterAddr:      c.clusterAddr,
		ClusterCert:      localCert,
		ClusterKeyParams: keyParams,
	}
	val, err := jsonutil.EncodeJSON(adv)
	if err != nil {
		return err
	}
	ent := &Entry{
		Key:   coreLeaderPrefix + uuid,
		Value: val,
	}
	err = c.barrier.Put(ctx, ent)
	if err != nil {
		return err
	}

	sd, ok := c.ha.(physical.ServiceDiscovery)
	if ok {
		if err := sd.NotifyActiveStateChange(); err != nil {
			if c.logger.IsWarn() {
				c.logger.Warn("core: failed to notify active status", "error", err)
			}
		}
	}
	return nil
}

func (c *Core) cleanLeaderPrefix(ctx context.Context, uuid string, leaderLostCh <-chan struct{}) {
	keys, err := c.barrier.List(ctx, coreLeaderPrefix)
	if err != nil {
		c.logger.Error("core: failed to list entries in core/leader", "error", err)
		return
	}
	for len(keys) > 0 {
		select {
		case <-time.After(leaderPrefixCleanDelay):
			if keys[0] != uuid {
				c.barrier.Delete(ctx, coreLeaderPrefix+keys[0])
			}
			keys = keys[1:]
		case <-leaderLostCh:
			return
		}
	}
}

// clearLeader is used to clear our leadership entry
func (c *Core) clearLeader(uuid string) error {
	key := coreLeaderPrefix + uuid
	err := c.barrier.Delete(c.activeContext, key)

	// Advertise ourselves as a standby
	sd, ok := c.ha.(physical.ServiceDiscovery)
	if ok {
		if err := sd.NotifyActiveStateChange(); err != nil {
			if c.logger.IsWarn() {
				c.logger.Warn("core: failed to notify standby status", "error", err)
			}
		}
	}

	return err
}

// emitMetrics is used to periodically expose metrics while runnig
func (c *Core) emitMetrics(stopCh chan struct{}) {
	for {
		select {
		case <-time.After(time.Second):
			c.metricsMutex.Lock()
			if c.expiration != nil {
				c.expiration.emitMetrics()
			}
			c.metricsMutex.Unlock()
		case <-stopCh:
			return
		}
	}
}

func (c *Core) ReplicationState() consts.ReplicationState {
	return consts.ReplicationState(atomic.LoadUint32(c.replicationState))
}

func (c *Core) ActiveNodeReplicationState() consts.ReplicationState {
	return consts.ReplicationState(atomic.LoadUint32(c.activeNodeReplicationState))
}

func (c *Core) SealAccess() *SealAccess {
	return NewSealAccess(c.seal)
}

func (c *Core) Logger() log.Logger {
	return c.logger
}

func (c *Core) BarrierKeyLength() (min, max int) {
	min, max = c.barrier.KeyLength()
	max += shamir.ShareOverhead
	return
}

func (c *Core) AuditedHeadersConfig() *AuditedHeadersConfig {
	return c.auditedHeaders
}

func lastRemoteWALImpl(c *Core) uint64 {
	return 0
}

func (c *Core) BarrierEncryptorAccess() *BarrierEncryptorAccess {
	return NewBarrierEncryptorAccess(c.barrier)
}

func (c *Core) PhysicalAccess() *physical.PhysicalAccess {
	return physical.NewPhysicalAccess(c.physical)
}

func (c *Core) RouterAccess() *RouterAccess {
	return NewRouterAccess(c)
}

// IsDRSecondary returns if the current cluster state is a DR secondary.
func (c *Core) IsDRSecondary() bool {
	return c.ReplicationState().HasState(consts.ReplicationDRSecondary)
}
