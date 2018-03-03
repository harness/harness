package vault

import (
	"testing"

	"github.com/hashicorp/vault/api"
	vaulthttp "github.com/hashicorp/vault/http"
	vaulttest "github.com/hashicorp/vault/vault"
)

// TestServer starts an in-memory Vault server configured with the provided
// core configuration and returns a pre-configured Vault client and closer func.
func TestServer(t *testing.T, conf *vaulttest.CoreConfig) (*api.Client, func()) {
	t.Helper()

	cluster := vaulttest.NewTestCluster(t, conf, &vaulttest.TestClusterOptions{
		HandlerFunc: vaulthttp.Handler,
		NumCores:    1,
	})
	cluster.Start()

	core := cluster.Cores[0].Core
	vaulttest.TestWaitActive(t, core)

	client := cluster.Cores[0].Client
	client.SetToken(cluster.RootToken)

	enableBackends(t, client, conf)

	return client, func() { defer cluster.Cleanup() }
}

// enableBackends takes a client for a Vault test server and a core configuration
// and enables/mounts all logical, credential, and audit backends listed in the
// configuration at their default paths. Listing the backends in the configuration
// is not enough; backends _must_ be enabled for API calls to work as expected
func enableBackends(t *testing.T, client *api.Client, conf *vaulttest.CoreConfig) {
	if conf != nil {
		if conf.LogicalBackends != nil {
			for k := range conf.LogicalBackends {
				err := client.Sys().Mount(k, &api.MountInput{
					Type: k,
				})
				if err != nil {
					t.Fatal(err)
				}
			}
		}

		if conf.CredentialBackends != nil {
			for k := range conf.CredentialBackends {
				err := client.Sys().EnableAuthWithOptions(k, &api.EnableAuthOptions{
					Type: k,
				})
				if err != nil {
					t.Fatal(err)
				}
			}
		}

		if conf.AuditBackends != nil {
			for k := range conf.AuditBackends {
				err := client.Sys().EnableAuditWithOptions(k, &api.EnableAuditOptions{
					Type: k,
				})
				if err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}
