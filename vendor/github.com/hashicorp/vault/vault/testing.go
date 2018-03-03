package vault

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	mathrand "math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	log "github.com/mgutz/logxi/v1"
	"github.com/mitchellh/copystructure"

	"golang.org/x/crypto/ssh"
	"golang.org/x/net/http2"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/audit"
	"github.com/hashicorp/vault/helper/logbridge"
	"github.com/hashicorp/vault/helper/logformat"
	"github.com/hashicorp/vault/helper/reload"
	"github.com/hashicorp/vault/helper/salt"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
	"github.com/hashicorp/vault/physical"
	"github.com/mitchellh/go-testing-interface"

	physInmem "github.com/hashicorp/vault/physical/inmem"
)

// This file contains a number of methods that are useful for unit
// tests within other packages.

const (
	testSharedPublicKey = `
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC9i+hFxZHGo6KblVme4zrAcJstR6I0PTJozW286X4WyvPnkMYDQ5mnhEYC7UWCvjoTWbPEXPX7NjhRtwQTGD67bV+lrxgfyzK1JZbUXK4PwgKJvQD+XyyWYMzDgGSQY61KUSqCxymSm/9NZkPU3ElaQ9xQuTzPpztM4ROfb8f2Yv6/ZESZsTo0MTAkp8Pcy+WkioI/uJ1H7zqs0EA4OMY4aDJRu0UtP4rTVeYNEAuRXdX+eH4aW3KMvhzpFTjMbaJHJXlEeUm2SaX5TNQyTOvghCeQILfYIL/Ca2ij8iwCmulwdV6eQGfd4VDu40PvSnmfoaE38o6HaPnX0kUcnKiT
`
	testSharedPrivateKey = `
-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEAvYvoRcWRxqOim5VZnuM6wHCbLUeiND0yaM1tvOl+Fsrz55DG
A0OZp4RGAu1Fgr46E1mzxFz1+zY4UbcEExg+u21fpa8YH8sytSWW1FyuD8ICib0A
/l8slmDMw4BkkGOtSlEqgscpkpv/TWZD1NxJWkPcULk8z6c7TOETn2/H9mL+v2RE
mbE6NDEwJKfD3MvlpIqCP7idR+86rNBAODjGOGgyUbtFLT+K01XmDRALkV3V/nh+
GltyjL4c6RU4zG2iRyV5RHlJtkml+UzUMkzr4IQnkCC32CC/wmtoo/IsAprpcHVe
nkBn3eFQ7uND70p5n6GhN/KOh2j519JFHJyokwIDAQABAoIBAHX7VOvBC3kCN9/x
+aPdup84OE7Z7MvpX6w+WlUhXVugnmsAAVDczhKoUc/WktLLx2huCGhsmKvyVuH+
MioUiE+vx75gm3qGx5xbtmOfALVMRLopjCnJYf6EaFA0ZeQ+NwowNW7Lu0PHmAU8
Z3JiX8IwxTz14DU82buDyewO7v+cEr97AnERe3PUcSTDoUXNaoNxjNpEJkKREY6h
4hAY676RT/GsRcQ8tqe/rnCqPHNd7JGqL+207FK4tJw7daoBjQyijWuB7K5chSal
oPInylM6b13ASXuOAOT/2uSUBWmFVCZPDCmnZxy2SdnJGbsJAMl7Ma3MUlaGvVI+
Tfh1aQkCgYEA4JlNOabTb3z42wz6mz+Nz3JRwbawD+PJXOk5JsSnV7DtPtfgkK9y
6FTQdhnozGWShAvJvc+C4QAihs9AlHXoaBY5bEU7R/8UK/pSqwzam+MmxmhVDV7G
IMQPV0FteoXTaJSikhZ88mETTegI2mik+zleBpVxvfdhE5TR+lq8Br0CgYEA2AwJ
CUD5CYUSj09PluR0HHqamWOrJkKPFPwa+5eiTTCzfBBxImYZh7nXnWuoviXC0sg2
AuvCW+uZ48ygv/D8gcz3j1JfbErKZJuV+TotK9rRtNIF5Ub7qysP7UjyI7zCssVM
kuDd9LfRXaB/qGAHNkcDA8NxmHW3gpln4CFdSY8CgYANs4xwfercHEWaJ1qKagAe
rZyrMpffAEhicJ/Z65lB0jtG4CiE6w8ZeUMWUVJQVcnwYD+4YpZbX4S7sJ0B8Ydy
AhkSr86D/92dKTIt2STk6aCN7gNyQ1vW198PtaAWH1/cO2UHgHOy3ZUt5X/Uwxl9
cex4flln+1Viumts2GgsCQKBgCJH7psgSyPekK5auFdKEr5+Gc/jB8I/Z3K9+g4X
5nH3G1PBTCJYLw7hRzw8W/8oALzvddqKzEFHphiGXK94Lqjt/A4q1OdbCrhiE68D
My21P/dAKB1UYRSs9Y8CNyHCjuZM9jSMJ8vv6vG/SOJPsnVDWVAckAbQDvlTHC9t
O98zAoGAcbW6uFDkrv0XMCpB9Su3KaNXOR0wzag+WIFQRXCcoTvxVi9iYfUReQPi
oOyBJU/HMVvBfv4g+OVFLVgSwwm6owwsouZ0+D/LasbuHqYyqYqdyPJQYzWA2Y+F
+B6f4RoPdSXj24JHPg/ioRxjaj094UXJxua2yfkcecGNEuBQHSs=
-----END RSA PRIVATE KEY-----
`
)

// TestCore returns a pure in-memory, uninitialized core for testing.
func TestCore(t testing.T) *Core {
	return TestCoreWithSeal(t, nil, false)
}

// TestCoreRaw returns a pure in-memory, uninitialized core for testing. The raw
// storage endpoints are enabled with this core.
func TestCoreRaw(t testing.T) *Core {
	return TestCoreWithSeal(t, nil, true)
}

// TestCoreNewSeal returns a pure in-memory, uninitialized core with
// the new seal configuration.
func TestCoreNewSeal(t testing.T) *Core {
	seal := NewTestSeal(t, nil)
	return TestCoreWithSeal(t, seal, false)
}

// TestCoreWithSeal returns a pure in-memory, uninitialized core with the
// specified seal for testing.
func TestCoreWithSeal(t testing.T, testSeal Seal, enableRaw bool) *Core {
	logger := logformat.NewVaultLogger(log.LevelTrace)
	physicalBackend, err := physInmem.NewInmem(nil, logger)
	if err != nil {
		t.Fatal(err)
	}

	conf := testCoreConfig(t, physicalBackend, logger)

	if enableRaw {
		conf.EnableRaw = true
	}

	if testSeal != nil {
		conf.Seal = testSeal
	}

	c, err := NewCore(conf)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return c
}

func testCoreConfig(t testing.T, physicalBackend physical.Backend, logger log.Logger) *CoreConfig {
	t.Helper()
	noopAudits := map[string]audit.Factory{
		"noop": func(_ context.Context, config *audit.BackendConfig) (audit.Backend, error) {
			view := &logical.InmemStorage{}
			view.Put(context.Background(), &logical.StorageEntry{
				Key:   "salt",
				Value: []byte("foo"),
			})
			config.SaltConfig = &salt.Config{
				HMAC:     sha256.New,
				HMACType: "hmac-sha256",
			}
			config.SaltView = view
			return &noopAudit{
				Config: config,
			}, nil
		},
	}

	noopBackends := make(map[string]logical.Factory)
	noopBackends["noop"] = func(ctx context.Context, config *logical.BackendConfig) (logical.Backend, error) {
		b := new(framework.Backend)
		b.Setup(ctx, config)
		return b, nil
	}
	noopBackends["http"] = func(ctx context.Context, config *logical.BackendConfig) (logical.Backend, error) {
		return new(rawHTTP), nil
	}

	credentialBackends := make(map[string]logical.Factory)
	for backendName, backendFactory := range noopBackends {
		credentialBackends[backendName] = backendFactory
	}
	for backendName, backendFactory := range testCredentialBackends {
		credentialBackends[backendName] = backendFactory
	}

	logicalBackends := make(map[string]logical.Factory)
	for backendName, backendFactory := range noopBackends {
		logicalBackends[backendName] = backendFactory
	}

	logicalBackends["kv"] = LeasedPassthroughBackendFactory
	for backendName, backendFactory := range testLogicalBackends {
		logicalBackends[backendName] = backendFactory
	}

	conf := &CoreConfig{
		Physical:           physicalBackend,
		AuditBackends:      noopAudits,
		LogicalBackends:    logicalBackends,
		CredentialBackends: credentialBackends,
		DisableMlock:       true,
		Logger:             logger,
	}

	return conf
}

// TestCoreInit initializes the core with a single key, and returns
// the key that must be used to unseal the core and a root token.
func TestCoreInit(t testing.T, core *Core) ([][]byte, string) {
	t.Helper()
	secretShares, _, root := TestCoreInitClusterWrapperSetup(t, core, nil, nil)
	return secretShares, root
}

func TestCoreInitClusterWrapperSetup(t testing.T, core *Core, clusterAddrs []*net.TCPAddr, handler http.Handler) ([][]byte, [][]byte, string) {
	t.Helper()
	core.SetClusterListenerAddrs(clusterAddrs)
	core.SetClusterHandler(handler)

	barrierConfig := &SealConfig{
		SecretShares:    3,
		SecretThreshold: 3,
	}

	// If we support storing barrier keys, then set that to equal the min threshold to unseal
	if core.seal.StoredKeysSupported() {
		barrierConfig.StoredShares = barrierConfig.SecretThreshold
	}

	recoveryConfig := &SealConfig{
		SecretShares:    3,
		SecretThreshold: 3,
	}

	result, err := core.Initialize(context.Background(), &InitParams{
		BarrierConfig:  barrierConfig,
		RecoveryConfig: recoveryConfig,
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	return result.SecretShares, result.RecoveryShares, result.RootToken
}

func TestCoreUnseal(core *Core, key []byte) (bool, error) {
	return core.Unseal(key)
}

func TestCoreUnsealWithRecoveryKeys(core *Core, key []byte) (bool, error) {
	return core.UnsealWithRecoveryKeys(context.Background(), key)
}

// TestCoreUnsealed returns a pure in-memory core that is already
// initialized and unsealed.
func TestCoreUnsealed(t testing.T) (*Core, [][]byte, string) {
	t.Helper()
	core := TestCore(t)
	return testCoreUnsealed(t, core)
}

// TestCoreUnsealedRaw returns a pure in-memory core that is already
// initialized, unsealed, and with raw endpoints enabled.
func TestCoreUnsealedRaw(t testing.T) (*Core, [][]byte, string) {
	t.Helper()
	core := TestCoreRaw(t)
	return testCoreUnsealed(t, core)
}

func testCoreUnsealed(t testing.T, core *Core) (*Core, [][]byte, string) {
	t.Helper()
	keys, token := TestCoreInit(t, core)
	for _, key := range keys {
		if _, err := TestCoreUnseal(core, TestKeyCopy(key)); err != nil {
			t.Fatalf("unseal err: %s", err)
		}
	}

	sealed, err := core.Sealed()
	if err != nil {
		t.Fatalf("err checking seal status: %s", err)
	}
	if sealed {
		t.Fatal("should not be sealed")
	}

	return core, keys, token
}

func TestCoreUnsealedBackend(t testing.T, backend physical.Backend) (*Core, [][]byte, string) {
	t.Helper()
	logger := logformat.NewVaultLogger(log.LevelTrace)
	conf := testCoreConfig(t, backend, logger)
	conf.Seal = NewTestSeal(t, nil)

	core, err := NewCore(conf)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	keys, token := TestCoreInit(t, core)
	for _, key := range keys {
		if _, err := TestCoreUnseal(core, TestKeyCopy(key)); err != nil {
			t.Fatalf("unseal err: %s", err)
		}
	}

	if err := core.UnsealWithStoredKeys(context.Background()); err != nil {
		t.Fatal(err)
	}

	sealed, err := core.Sealed()
	if err != nil {
		t.Fatalf("err checking seal status: %s", err)
	}
	if sealed {
		t.Fatal("should not be sealed")
	}

	return core, keys, token
}

func testTokenStore(t testing.T, c *Core) *TokenStore {
	me := &MountEntry{
		Table:       credentialTableType,
		Path:        "token/",
		Type:        "token",
		Description: "token based credentials",
	}

	meUUID, err := uuid.GenerateUUID()
	if err != nil {
		t.Fatal(err)
	}
	me.UUID = meUUID

	view := NewBarrierView(c.barrier, credentialBarrierPrefix+me.UUID+"/")
	sysView := c.mountEntrySysView(me)

	tokenstore, _ := c.newCredentialBackend(context.Background(), "token", sysView, view, nil)
	ts := tokenstore.(*TokenStore)

	err = c.router.Unmount(context.Background(), "auth/token/")
	if err != nil {
		t.Fatal(err)
	}
	err = c.router.Mount(ts, "auth/token/", &MountEntry{Table: credentialTableType, UUID: "authtokenuuid", Path: "auth/token", Accessor: "authtokenaccessor"}, ts.view)
	if err != nil {
		t.Fatal(err)
	}

	ts.SetExpirationManager(c.expiration)

	return ts
}

// TestCoreWithTokenStore returns an in-memory core that has a token store
// mounted, so that logical token functions can be used
func TestCoreWithTokenStore(t testing.T) (*Core, *TokenStore, [][]byte, string) {
	c, keys, root := TestCoreUnsealed(t)

	return c, c.tokenStore, keys, root
}

// TestCoreWithBackendTokenStore returns a core that has a token store
// mounted and used the provided physical backend, so that logical token
// functions can be used
func TestCoreWithBackendTokenStore(t testing.T, backend physical.Backend) (*Core, *TokenStore, [][]byte, string) {
	c, keys, root := TestCoreUnsealedBackend(t, backend)
	ts := testTokenStore(t, c)

	return c, ts, keys, root
}

// TestKeyCopy is a silly little function to just copy the key so that
// it can be used with Unseal easily.
func TestKeyCopy(key []byte) []byte {
	result := make([]byte, len(key))
	copy(result, key)
	return result
}

func TestDynamicSystemView(c *Core) *dynamicSystemView {
	me := &MountEntry{
		Config: MountConfig{
			DefaultLeaseTTL: 24 * time.Hour,
			MaxLeaseTTL:     2 * 24 * time.Hour,
		},
	}

	return &dynamicSystemView{c, me}
}

// TestAddTestPlugin registers the testFunc as part of the plugin command to the
// plugin catalog.
func TestAddTestPlugin(t testing.T, c *Core, name, testFunc string) {
	file, err := os.Open(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	hash := sha256.New()

	_, err = io.Copy(hash, file)
	if err != nil {
		t.Fatal(err)
	}

	sum := hash.Sum(nil)

	// Determine plugin directory path
	fullPath, err := filepath.EvalSymlinks(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	directoryPath := filepath.Dir(fullPath)

	// Set core's plugin directory and plugin catalog directory
	c.pluginDirectory = directoryPath
	c.pluginCatalog.directory = directoryPath

	command := fmt.Sprintf("%s", filepath.Base(os.Args[0]))
	args := []string{fmt.Sprintf("--test.run=%s", testFunc)}
	err = c.pluginCatalog.Set(context.Background(), name, command, args, sum)
	if err != nil {
		t.Fatal(err)
	}
}

// TestAddTestPluginTempDir registers the testFunc as part of the plugin command to the
// plugin catalog. It uses tmpDir as the plugin directory.
func TestAddTestPluginTempDir(t testing.T, c *Core, name, testFunc, tempDir string) {
	file, err := os.Open(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}

	// Copy over the file to the temp dir
	dst := filepath.Join(tempDir, filepath.Base(os.Args[0]))
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fi.Mode())
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	if _, err = io.Copy(out, file); err != nil {
		t.Fatal(err)
	}
	err = out.Sync()
	if err != nil {
		t.Fatal(err)
	}

	// Determine plugin directory full path
	fullPath, err := filepath.EvalSymlinks(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	reader, err := os.Open(filepath.Join(fullPath, filepath.Base(os.Args[0])))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	// Find out the sha256
	hash := sha256.New()

	_, err = io.Copy(hash, reader)
	if err != nil {
		t.Fatal(err)
	}

	sum := hash.Sum(nil)

	// Set core's plugin directory and plugin catalog directory
	c.pluginDirectory = fullPath
	c.pluginCatalog.directory = fullPath

	command := fmt.Sprintf("%s", filepath.Base(os.Args[0]))
	args := []string{fmt.Sprintf("--test.run=%s", testFunc)}
	err = c.pluginCatalog.Set(context.Background(), name, command, args, sum)
	if err != nil {
		t.Fatal(err)
	}
}

var testLogicalBackends = map[string]logical.Factory{}
var testCredentialBackends = map[string]logical.Factory{}

// StartSSHHostTestServer starts the test server which responds to SSH
// authentication. Used to test the SSH secret backend.
func StartSSHHostTestServer() (string, error) {
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(testSharedPublicKey))
	if err != nil {
		return "", fmt.Errorf("Error parsing public key")
	}
	serverConfig := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if bytes.Compare(pubKey.Marshal(), key.Marshal()) == 0 {
				return &ssh.Permissions{}, nil
			} else {
				return nil, fmt.Errorf("Key does not match")
			}
		},
	}
	signer, err := ssh.ParsePrivateKey([]byte(testSharedPrivateKey))
	if err != nil {
		panic("Error parsing private key")
	}
	serverConfig.AddHostKey(signer)

	soc, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("Error listening to connection")
	}

	go func() {
		for {
			conn, err := soc.Accept()
			if err != nil {
				panic(fmt.Sprintf("Error accepting incoming connection: %s", err))
			}
			defer conn.Close()
			sshConn, chanReqs, _, err := ssh.NewServerConn(conn, serverConfig)
			if err != nil {
				panic(fmt.Sprintf("Handshaking error: %v", err))
			}

			go func() {
				for chanReq := range chanReqs {
					go func(chanReq ssh.NewChannel) {
						if chanReq.ChannelType() != "session" {
							chanReq.Reject(ssh.UnknownChannelType, "unknown channel type")
							return
						}

						ch, requests, err := chanReq.Accept()
						if err != nil {
							panic(fmt.Sprintf("Error accepting channel: %s", err))
						}

						go func(ch ssh.Channel, in <-chan *ssh.Request) {
							for req := range in {
								executeServerCommand(ch, req)
							}
						}(ch, requests)
					}(chanReq)
				}
				sshConn.Close()
			}()
		}
	}()
	return soc.Addr().String(), nil
}

// This executes the commands requested to be run on the server.
// Used to test the SSH secret backend.
func executeServerCommand(ch ssh.Channel, req *ssh.Request) {
	command := string(req.Payload[4:])
	cmd := exec.Command("/bin/bash", []string{"-c", command}...)
	req.Reply(true, nil)

	cmd.Stdout = ch
	cmd.Stderr = ch
	cmd.Stdin = ch

	err := cmd.Start()
	if err != nil {
		panic(fmt.Sprintf("Error starting the command: '%s'", err))
	}

	go func() {
		_, err := cmd.Process.Wait()
		if err != nil {
			panic(fmt.Sprintf("Error while waiting for command to finish:'%s'", err))
		}
		ch.Close()
	}()
}

// This adds a credential backend for the test core. This needs to be
// invoked before the test core is created.
func AddTestCredentialBackend(name string, factory logical.Factory) error {
	if name == "" {
		return fmt.Errorf("missing backend name")
	}
	if factory == nil {
		return fmt.Errorf("missing backend factory function")
	}
	testCredentialBackends[name] = factory
	return nil
}

// This adds a logical backend for the test core. This needs to be
// invoked before the test core is created.
func AddTestLogicalBackend(name string, factory logical.Factory) error {
	if name == "" {
		return fmt.Errorf("Missing backend name")
	}
	if factory == nil {
		return fmt.Errorf("Missing backend factory function")
	}
	testLogicalBackends[name] = factory
	return nil
}

type noopAudit struct {
	Config    *audit.BackendConfig
	salt      *salt.Salt
	saltMutex sync.RWMutex
}

func (n *noopAudit) GetHash(data string) (string, error) {
	salt, err := n.Salt()
	if err != nil {
		return "", err
	}
	return salt.GetIdentifiedHMAC(data), nil
}

func (n *noopAudit) LogRequest(_ context.Context, _ *audit.LogInput) error {
	return nil
}

func (n *noopAudit) LogResponse(_ context.Context, _ *audit.LogInput) error {
	return nil
}

func (n *noopAudit) Reload(_ context.Context) error {
	return nil
}

func (n *noopAudit) Invalidate(_ context.Context) {
	n.saltMutex.Lock()
	defer n.saltMutex.Unlock()
	n.salt = nil
}

func (n *noopAudit) Salt() (*salt.Salt, error) {
	n.saltMutex.RLock()
	if n.salt != nil {
		defer n.saltMutex.RUnlock()
		return n.salt, nil
	}
	n.saltMutex.RUnlock()
	n.saltMutex.Lock()
	defer n.saltMutex.Unlock()
	if n.salt != nil {
		return n.salt, nil
	}
	salt, err := salt.NewSalt(n.Config.SaltView, n.Config.SaltConfig)
	if err != nil {
		return nil, err
	}
	n.salt = salt
	return salt, nil
}

type rawHTTP struct{}

func (n *rawHTTP) HandleRequest(ctx context.Context, req *logical.Request) (*logical.Response, error) {
	return &logical.Response{
		Data: map[string]interface{}{
			logical.HTTPStatusCode:  200,
			logical.HTTPContentType: "plain/text",
			logical.HTTPRawBody:     []byte("hello world"),
		},
	}, nil
}

func (n *rawHTTP) HandleExistenceCheck(ctx context.Context, req *logical.Request) (bool, bool, error) {
	return false, false, nil
}

func (n *rawHTTP) SpecialPaths() *logical.Paths {
	return &logical.Paths{Unauthenticated: []string{"*"}}
}

func (n *rawHTTP) System() logical.SystemView {
	return logical.StaticSystemView{
		DefaultLeaseTTLVal: time.Hour * 24,
		MaxLeaseTTLVal:     time.Hour * 24 * 32,
	}
}

func (n *rawHTTP) Logger() log.Logger {
	return logformat.NewVaultLogger(log.LevelTrace)
}

func (n *rawHTTP) Cleanup(ctx context.Context) {
	// noop
}

func (n *rawHTTP) Initialize(ctx context.Context) error {
	// noop
	return nil
}

func (n *rawHTTP) InvalidateKey(context.Context, string) {
	// noop
}

func (n *rawHTTP) Setup(ctx context.Context, config *logical.BackendConfig) error {
	// noop
	return nil
}

func (n *rawHTTP) Type() logical.BackendType {
	return logical.TypeUnknown
}

func GenerateRandBytes(length int) ([]byte, error) {
	if length < 0 {
		return nil, fmt.Errorf("length must be >= 0")
	}

	buf := make([]byte, length)
	if length == 0 {
		return buf, nil
	}

	n, err := rand.Read(buf)
	if err != nil {
		return nil, err
	}
	if n != length {
		return nil, fmt.Errorf("unable to read %d bytes; only read %d", length, n)
	}

	return buf, nil
}

func TestWaitActive(t testing.T, core *Core) {
	t.Helper()
	start := time.Now()
	var standby bool
	var err error
	for time.Now().Sub(start) < time.Second {
		standby, err = core.Standby()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if !standby {
			break
		}
	}
	if standby {
		t.Fatalf("should not be in standby mode")
	}
}

type TestCluster struct {
	BarrierKeys   [][]byte
	RecoveryKeys  [][]byte
	CACert        *x509.Certificate
	CACertBytes   []byte
	CACertPEM     []byte
	CACertPEMFile string
	CAKey         *ecdsa.PrivateKey
	CAKeyPEM      []byte
	Cores         []*TestClusterCore
	ID            string
	RootToken     string
	RootCAs       *x509.CertPool
	TempDir       string
}

func (c *TestCluster) Start() {
	for _, core := range c.Cores {
		if core.Server != nil {
			for _, ln := range core.Listeners {
				go core.Server.Serve(ln)
			}
		}
	}
}

func (c *TestCluster) EnsureCoresSealed(t testing.T) {
	t.Helper()
	if err := c.ensureCoresSealed(); err != nil {
		t.Fatal(err)
	}
}

func (c *TestCluster) Cleanup() {
	// Close listeners
	for _, core := range c.Cores {
		if core.Listeners != nil {
			for _, ln := range core.Listeners {
				ln.Close()
			}
		}
	}

	// Seal the cores
	c.ensureCoresSealed()

	// Remove any temp dir that exists
	if c.TempDir != "" {
		os.RemoveAll(c.TempDir)
	}

	// Give time to actually shut down/clean up before the next test
	time.Sleep(time.Second)
}

func (c *TestCluster) ensureCoresSealed() error {
	for _, core := range c.Cores {
		if err := core.Shutdown(); err != nil {
			return err
		}
		timeout := time.Now().Add(60 * time.Second)
		for {
			if time.Now().After(timeout) {
				return fmt.Errorf("timeout waiting for core to seal")
			}
			sealed, err := core.Sealed()
			if err != nil {
				return err
			}
			if sealed {
				break
			}
			time.Sleep(250 * time.Millisecond)
		}
	}
	return nil
}

// UnsealWithStoredKeys uses stored keys to unseal the test cluster cores
func (c *TestCluster) UnsealWithStoredKeys(t testing.T) error {
	for _, core := range c.Cores {
		if err := core.UnsealWithStoredKeys(context.Background()); err != nil {
			return err
		}
		timeout := time.Now().Add(60 * time.Second)
		for {
			if time.Now().After(timeout) {
				return fmt.Errorf("timeout waiting for core to unseal")
			}
			sealed, err := core.Sealed()
			if err != nil {
				return err
			}
			if !sealed {
				break
			}
			time.Sleep(250 * time.Millisecond)
		}
	}
	return nil
}

type TestListener struct {
	net.Listener
	Address *net.TCPAddr
}

type TestClusterCore struct {
	*Core
	Client            *api.Client
	Handler           http.Handler
	Listeners         []*TestListener
	ReloadFuncs       *map[string][]reload.ReloadFunc
	ReloadFuncsLock   *sync.RWMutex
	Server            *http.Server
	ServerCert        *x509.Certificate
	ServerCertBytes   []byte
	ServerCertPEM     []byte
	ServerKey         *ecdsa.PrivateKey
	ServerKeyPEM      []byte
	TLSConfig         *tls.Config
	UnderlyingStorage physical.Backend
}

type TestClusterOptions struct {
	KeepStandbysSealed bool
	SkipInit           bool
	HandlerFunc        func(*Core) http.Handler
	BaseListenAddress  string
	NumCores           int
	SealFunc           func() Seal
	RawLogger          interface{}
	TempDir            string
	CACert             []byte
	CAKey              *ecdsa.PrivateKey
}

var DefaultNumCores = 3

type certInfo struct {
	cert      *x509.Certificate
	certPEM   []byte
	certBytes []byte
	key       *ecdsa.PrivateKey
	keyPEM    []byte
}

// NewTestCluster creates a new test cluster based on the provided core config
// and test cluster options.
//
// N.B. Even though a single base CoreConfig is provided, NewTestCluster will instantiate a
// core config for each core it creates. If separate seal per core is desired, opts.SealFunc
// can be provided to generate a seal for each one. Otherwise, the provided base.Seal will be
// shared among cores. NewCore's default behavior is to generate a new DefaultSeal if the
// provided Seal in coreConfig (i.e. base.Seal) is nil.
func NewTestCluster(t testing.T, base *CoreConfig, opts *TestClusterOptions) *TestCluster {
	var err error

	var numCores int
	if opts == nil || opts.NumCores == 0 {
		numCores = DefaultNumCores
	} else {
		numCores = opts.NumCores
	}

	certIPs := []net.IP{
		net.IPv6loopback,
		net.ParseIP("127.0.0.1"),
	}
	var baseAddr *net.TCPAddr
	if opts != nil && opts.BaseListenAddress != "" {
		baseAddr, err = net.ResolveTCPAddr("tcp", opts.BaseListenAddress)
		if err != nil {
			t.Fatal("could not parse given base IP")
		}
		certIPs = append(certIPs, baseAddr.IP)
	}

	var testCluster TestCluster
	if opts != nil && opts.TempDir != "" {
		if _, err := os.Stat(opts.TempDir); os.IsNotExist(err) {
			if err := os.MkdirAll(opts.TempDir, 0700); err != nil {
				t.Fatal(err)
			}
		}
		testCluster.TempDir = opts.TempDir
	} else {
		tempDir, err := ioutil.TempDir("", "vault-test-cluster-")
		if err != nil {
			t.Fatal(err)
		}
		testCluster.TempDir = tempDir
	}

	var caKey *ecdsa.PrivateKey
	if opts != nil && opts.CAKey != nil {
		caKey = opts.CAKey
	} else {
		caKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
	}
	testCluster.CAKey = caKey
	var caBytes []byte
	if opts != nil && len(opts.CACert) > 0 {
		caBytes = opts.CACert
	} else {
		caCertTemplate := &x509.Certificate{
			Subject: pkix.Name{
				CommonName: "localhost",
			},
			DNSNames:              []string{"localhost"},
			IPAddresses:           certIPs,
			KeyUsage:              x509.KeyUsage(x509.KeyUsageCertSign | x509.KeyUsageCRLSign),
			SerialNumber:          big.NewInt(mathrand.Int63()),
			NotBefore:             time.Now().Add(-30 * time.Second),
			NotAfter:              time.Now().Add(262980 * time.Hour),
			BasicConstraintsValid: true,
			IsCA: true,
		}
		caBytes, err = x509.CreateCertificate(rand.Reader, caCertTemplate, caCertTemplate, caKey.Public(), caKey)
		if err != nil {
			t.Fatal(err)
		}
	}
	caCert, err := x509.ParseCertificate(caBytes)
	if err != nil {
		t.Fatal(err)
	}
	testCluster.CACert = caCert
	testCluster.CACertBytes = caBytes
	testCluster.RootCAs = x509.NewCertPool()
	testCluster.RootCAs.AddCert(caCert)
	caCertPEMBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	}
	testCluster.CACertPEM = pem.EncodeToMemory(caCertPEMBlock)
	testCluster.CACertPEMFile = filepath.Join(testCluster.TempDir, "ca_cert.pem")
	err = ioutil.WriteFile(testCluster.CACertPEMFile, testCluster.CACertPEM, 0755)
	if err != nil {
		t.Fatal(err)
	}
	marshaledCAKey, err := x509.MarshalECPrivateKey(caKey)
	if err != nil {
		t.Fatal(err)
	}
	caKeyPEMBlock := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: marshaledCAKey,
	}
	testCluster.CAKeyPEM = pem.EncodeToMemory(caKeyPEMBlock)
	err = ioutil.WriteFile(filepath.Join(testCluster.TempDir, "ca_key.pem"), testCluster.CAKeyPEM, 0755)
	if err != nil {
		t.Fatal(err)
	}

	var certInfoSlice []*certInfo

	//
	// Certs generation
	//
	for i := 0; i < numCores; i++ {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		certTemplate := &x509.Certificate{
			Subject: pkix.Name{
				CommonName: "localhost",
			},
			DNSNames:    []string{"localhost"},
			IPAddresses: certIPs,
			ExtKeyUsage: []x509.ExtKeyUsage{
				x509.ExtKeyUsageServerAuth,
				x509.ExtKeyUsageClientAuth,
			},
			KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageKeyAgreement,
			SerialNumber: big.NewInt(mathrand.Int63()),
			NotBefore:    time.Now().Add(-30 * time.Second),
			NotAfter:     time.Now().Add(262980 * time.Hour),
		}
		certBytes, err := x509.CreateCertificate(rand.Reader, certTemplate, caCert, key.Public(), caKey)
		if err != nil {
			t.Fatal(err)
		}
		cert, err := x509.ParseCertificate(certBytes)
		if err != nil {
			t.Fatal(err)
		}
		certPEMBlock := &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certBytes,
		}
		certPEM := pem.EncodeToMemory(certPEMBlock)
		marshaledKey, err := x509.MarshalECPrivateKey(key)
		if err != nil {
			t.Fatal(err)
		}
		keyPEMBlock := &pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: marshaledKey,
		}
		keyPEM := pem.EncodeToMemory(keyPEMBlock)

		certInfoSlice = append(certInfoSlice, &certInfo{
			cert:      cert,
			certPEM:   certPEM,
			certBytes: certBytes,
			key:       key,
			keyPEM:    keyPEM,
		})
	}

	//
	// Listener setup
	//
	logger := logformat.NewVaultLogger(log.LevelTrace)
	ports := make([]int, numCores)
	if baseAddr != nil {
		for i := 0; i < numCores; i++ {
			ports[i] = baseAddr.Port + i
		}
	} else {
		baseAddr = &net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 0,
		}
	}

	listeners := [][]*TestListener{}
	servers := []*http.Server{}
	handlers := []http.Handler{}
	tlsConfigs := []*tls.Config{}
	certGetters := []*reload.CertificateGetter{}
	for i := 0; i < numCores; i++ {
		baseAddr.Port = ports[i]
		ln, err := net.ListenTCP("tcp", baseAddr)
		if err != nil {
			t.Fatal(err)
		}
		certFile := filepath.Join(testCluster.TempDir, fmt.Sprintf("node%d_port_%d_cert.pem", i+1, ln.Addr().(*net.TCPAddr).Port))
		keyFile := filepath.Join(testCluster.TempDir, fmt.Sprintf("node%d_port_%d_key.pem", i+1, ln.Addr().(*net.TCPAddr).Port))
		err = ioutil.WriteFile(certFile, certInfoSlice[i].certPEM, 0755)
		if err != nil {
			t.Fatal(err)
		}
		err = ioutil.WriteFile(keyFile, certInfoSlice[i].keyPEM, 0755)
		if err != nil {
			t.Fatal(err)
		}
		tlsCert, err := tls.X509KeyPair(certInfoSlice[i].certPEM, certInfoSlice[i].keyPEM)
		if err != nil {
			t.Fatal(err)
		}
		certGetter := reload.NewCertificateGetter(certFile, keyFile, "")
		certGetters = append(certGetters, certGetter)
		tlsConfig := &tls.Config{
			Certificates:   []tls.Certificate{tlsCert},
			RootCAs:        testCluster.RootCAs,
			ClientCAs:      testCluster.RootCAs,
			ClientAuth:     tls.RequestClientCert,
			NextProtos:     []string{"h2", "http/1.1"},
			GetCertificate: certGetter.GetCertificate,
		}
		tlsConfig.BuildNameToCertificate()
		tlsConfigs = append(tlsConfigs, tlsConfig)
		lns := []*TestListener{&TestListener{
			Listener: tls.NewListener(ln, tlsConfig),
			Address:  ln.Addr().(*net.TCPAddr),
		},
		}
		listeners = append(listeners, lns)
		var handler http.Handler = http.NewServeMux()
		handlers = append(handlers, handler)
		server := &http.Server{
			Handler: handler,
		}
		servers = append(servers, server)
	}

	// Create three cores with the same physical and different redirect/cluster
	// addrs.
	// N.B.: On OSX, instead of random ports, it assigns new ports to new
	// listeners sequentially. Aside from being a bad idea in a security sense,
	// it also broke tests that assumed it was OK to just use the port above
	// the redirect addr. This has now been changed to 105 ports above, but if
	// we ever do more than three nodes in a cluster it may need to be bumped.
	// Note: it's 105 so that we don't conflict with a running Consul by
	// default.
	coreConfig := &CoreConfig{
		LogicalBackends:    make(map[string]logical.Factory),
		CredentialBackends: make(map[string]logical.Factory),
		AuditBackends:      make(map[string]audit.Factory),
		RedirectAddr:       fmt.Sprintf("https://127.0.0.1:%d", listeners[0][0].Address.Port),
		ClusterAddr:        fmt.Sprintf("https://127.0.0.1:%d", listeners[0][0].Address.Port+105),
		DisableMlock:       true,
		EnableUI:           true,
	}

	if base != nil {
		coreConfig.DisableCache = base.DisableCache
		coreConfig.EnableUI = base.EnableUI
		coreConfig.DefaultLeaseTTL = base.DefaultLeaseTTL
		coreConfig.MaxLeaseTTL = base.MaxLeaseTTL
		coreConfig.CacheSize = base.CacheSize
		coreConfig.PluginDirectory = base.PluginDirectory
		coreConfig.Seal = base.Seal
		coreConfig.DevToken = base.DevToken

		if !coreConfig.DisableMlock {
			base.DisableMlock = false
		}

		if base.Physical != nil {
			coreConfig.Physical = base.Physical
		}

		if base.HAPhysical != nil {
			coreConfig.HAPhysical = base.HAPhysical
		}

		// Used to set something non-working to test fallback
		switch base.ClusterAddr {
		case "empty":
			coreConfig.ClusterAddr = ""
		case "":
		default:
			coreConfig.ClusterAddr = base.ClusterAddr
		}

		if base.LogicalBackends != nil {
			for k, v := range base.LogicalBackends {
				coreConfig.LogicalBackends[k] = v
			}
		}
		if base.CredentialBackends != nil {
			for k, v := range base.CredentialBackends {
				coreConfig.CredentialBackends[k] = v
			}
		}
		if base.AuditBackends != nil {
			for k, v := range base.AuditBackends {
				coreConfig.AuditBackends[k] = v
			}
		}
		if base.Logger != nil {
			coreConfig.Logger = base.Logger
		}

		coreConfig.ClusterCipherSuites = base.ClusterCipherSuites

		coreConfig.DisableCache = base.DisableCache

		coreConfig.DevToken = base.DevToken
	}

	if coreConfig.Physical == nil {
		coreConfig.Physical, err = physInmem.NewInmem(nil, logger)
		if err != nil {
			t.Fatal(err)
		}
	}
	if coreConfig.HAPhysical == nil {
		haPhys, err := physInmem.NewInmemHA(nil, logger)
		if err != nil {
			t.Fatal(err)
		}
		coreConfig.HAPhysical = haPhys.(physical.HABackend)
	}

	cores := []*Core{}
	for i := 0; i < numCores; i++ {
		coreConfig.RedirectAddr = fmt.Sprintf("https://127.0.0.1:%d", listeners[i][0].Address.Port)
		if coreConfig.ClusterAddr != "" {
			coreConfig.ClusterAddr = fmt.Sprintf("https://127.0.0.1:%d", listeners[i][0].Address.Port+105)
		}

		// if opts.SealFunc is provided, use that to generate a seal for the config instead
		if opts != nil && opts.SealFunc != nil {
			coreConfig.Seal = opts.SealFunc()
		}

		if opts != nil && opts.RawLogger != nil {
			switch opts.RawLogger.(type) {
			case *logbridge.Logger:
				coreConfig.Logger = opts.RawLogger.(*logbridge.Logger).Named(fmt.Sprintf("core%d", i)).LogxiLogger()
			case *logbridge.LogxiLogger:
				coreConfig.Logger = opts.RawLogger.(*logbridge.LogxiLogger).Named(fmt.Sprintf("core%d", i))
			}
		}

		c, err := NewCore(coreConfig)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		cores = append(cores, c)
		if opts != nil && opts.HandlerFunc != nil {
			handlers[i] = opts.HandlerFunc(c)
			servers[i].Handler = handlers[i]
		}
	}

	//
	// Clustering setup
	//
	clusterAddrGen := func(lns []*TestListener) []*net.TCPAddr {
		ret := make([]*net.TCPAddr, len(lns))
		for i, ln := range lns {
			ret[i] = &net.TCPAddr{
				IP:   ln.Address.IP,
				Port: ln.Address.Port + 105,
			}
		}
		return ret
	}

	if numCores > 1 {
		for i := 1; i < numCores; i++ {
			cores[i].SetClusterListenerAddrs(clusterAddrGen(listeners[i]))
			cores[i].SetClusterHandler(handlers[i])
		}
	}

	if opts == nil || !opts.SkipInit {
		bKeys, rKeys, root := TestCoreInitClusterWrapperSetup(t, cores[0], clusterAddrGen(listeners[0]), handlers[0])
		barrierKeys, _ := copystructure.Copy(bKeys)
		testCluster.BarrierKeys = barrierKeys.([][]byte)
		recoveryKeys, _ := copystructure.Copy(rKeys)
		testCluster.RecoveryKeys = recoveryKeys.([][]byte)
		testCluster.RootToken = root

		// Write root token and barrier keys
		err = ioutil.WriteFile(filepath.Join(testCluster.TempDir, "root_token"), []byte(root), 0755)
		if err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		for i, key := range testCluster.BarrierKeys {
			buf.Write([]byte(base64.StdEncoding.EncodeToString(key)))
			if i < len(testCluster.BarrierKeys)-1 {
				buf.WriteRune('\n')
			}
		}
		err = ioutil.WriteFile(filepath.Join(testCluster.TempDir, "barrier_keys"), buf.Bytes(), 0755)
		if err != nil {
			t.Fatal(err)
		}
		for i, key := range testCluster.RecoveryKeys {
			buf.Write([]byte(base64.StdEncoding.EncodeToString(key)))
			if i < len(testCluster.RecoveryKeys)-1 {
				buf.WriteRune('\n')
			}
		}
		err = ioutil.WriteFile(filepath.Join(testCluster.TempDir, "recovery_keys"), buf.Bytes(), 0755)
		if err != nil {
			t.Fatal(err)
		}

		// Unseal first core
		for _, key := range bKeys {
			if _, err := cores[0].Unseal(TestKeyCopy(key)); err != nil {
				t.Fatalf("unseal err: %s", err)
			}
		}

		ctx := context.Background()

		// If stored keys is supported, the above will no no-op, so trigger auto-unseal
		// using stored keys to try to unseal
		if err := cores[0].UnsealWithStoredKeys(ctx); err != nil {
			t.Fatal(err)
		}

		// Verify unsealed
		sealed, err := cores[0].Sealed()
		if err != nil {
			t.Fatalf("err checking seal status: %s", err)
		}
		if sealed {
			t.Fatal("should not be sealed")
		}

		TestWaitActive(t, cores[0])

		// Unseal other cores unless otherwise specified
		if (opts == nil || !opts.KeepStandbysSealed) && numCores > 1 {
			for i := 1; i < numCores; i++ {
				for _, key := range bKeys {
					if _, err := cores[i].Unseal(TestKeyCopy(key)); err != nil {
						t.Fatalf("unseal err: %s", err)
					}
				}

				// If stored keys is supported, the above will no no-op, so trigger auto-unseal
				// using stored keys
				if err := cores[i].UnsealWithStoredKeys(ctx); err != nil {
					t.Fatal(err)
				}
			}

			// Let them come fully up to standby
			time.Sleep(2 * time.Second)

			// Ensure cluster connection info is populated.
			// Other cores should not come up as leaders.
			for i := 1; i < numCores; i++ {
				isLeader, _, _, err := cores[i].Leader()
				if err != nil {
					t.Fatal(err)
				}
				if isLeader {
					t.Fatalf("core[%d] should not be leader", i)
				}
			}
		}

		//
		// Set test cluster core(s) and test cluster
		//
		cluster, err := cores[0].Cluster(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		testCluster.ID = cluster.ID
	}

	getAPIClient := func(port int, tlsConfig *tls.Config) *api.Client {
		transport := cleanhttp.DefaultPooledTransport()
		transport.TLSClientConfig = tlsConfig.Clone()
		if err := http2.ConfigureTransport(transport); err != nil {
			t.Fatal(err)
		}
		client := &http.Client{
			Transport: transport,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				// This can of course be overridden per-test by using its own client
				return fmt.Errorf("redirects not allowed in these tests")
			},
		}
		config := api.DefaultConfig()
		if config.Error != nil {
			t.Fatal(config.Error)
		}
		config.Address = fmt.Sprintf("https://127.0.0.1:%d", port)
		config.HttpClient = client
		apiClient, err := api.NewClient(config)
		if err != nil {
			t.Fatal(err)
		}
		if opts == nil || !opts.SkipInit {
			apiClient.SetToken(testCluster.RootToken)
		}
		return apiClient
	}

	var ret []*TestClusterCore
	for i := 0; i < numCores; i++ {
		tcc := &TestClusterCore{
			Core:            cores[i],
			ServerKey:       certInfoSlice[i].key,
			ServerKeyPEM:    certInfoSlice[i].keyPEM,
			ServerCert:      certInfoSlice[i].cert,
			ServerCertBytes: certInfoSlice[i].certBytes,
			ServerCertPEM:   certInfoSlice[i].certPEM,
			Listeners:       listeners[i],
			Handler:         handlers[i],
			Server:          servers[i],
			TLSConfig:       tlsConfigs[i],
			Client:          getAPIClient(listeners[i][0].Address.Port, tlsConfigs[i]),
		}
		tcc.ReloadFuncs = &cores[i].reloadFuncs
		tcc.ReloadFuncsLock = &cores[i].reloadFuncsLock
		tcc.ReloadFuncsLock.Lock()
		(*tcc.ReloadFuncs)["listener|tcp"] = []reload.ReloadFunc{certGetters[i].Reload}
		tcc.ReloadFuncsLock.Unlock()
		ret = append(ret, tcc)
	}

	testCluster.Cores = ret
	return &testCluster
}
