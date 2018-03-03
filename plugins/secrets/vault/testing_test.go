package vault

import (
	"testing"

	"github.com/hashicorp/vault/audit"
	"github.com/hashicorp/vault/builtin/audit/file"
	"github.com/hashicorp/vault/builtin/audit/socket"
	"github.com/hashicorp/vault/builtin/audit/syslog"
	"github.com/hashicorp/vault/builtin/credential/approle"
	"github.com/hashicorp/vault/builtin/credential/cert"
	"github.com/hashicorp/vault/builtin/credential/github"
	"github.com/hashicorp/vault/builtin/credential/ldap"
	"github.com/hashicorp/vault/builtin/credential/okta"
	"github.com/hashicorp/vault/builtin/credential/radius"
	"github.com/hashicorp/vault/builtin/credential/userpass"
	"github.com/hashicorp/vault/builtin/logical/aws"
	"github.com/hashicorp/vault/builtin/logical/cassandra"
	"github.com/hashicorp/vault/builtin/logical/consul"
	"github.com/hashicorp/vault/builtin/logical/database"
	"github.com/hashicorp/vault/builtin/logical/nomad"
	"github.com/hashicorp/vault/builtin/logical/pki"
	"github.com/hashicorp/vault/builtin/logical/rabbitmq"
	"github.com/hashicorp/vault/builtin/logical/ssh"
	"github.com/hashicorp/vault/builtin/logical/totp"
	"github.com/hashicorp/vault/builtin/logical/transit"
	"github.com/hashicorp/vault/logical"
	vaulttest "github.com/hashicorp/vault/vault"
)

// Tests that passing a nil CoreConfig will result in a test Vault server with
// all of the default backends present (secret/ in particular)
func TestTestServer(t *testing.T) {
	client, closer := TestServer(t, nil)
	defer closer()

	path := "secret/foo"
	data := map[string]interface{}{
		"value": "bar",
	}

	_, err := client.Logical().Write(path, data)
	if err != nil {
		t.Fatal(err)
	}

	secret, err := client.Logical().Read(path)
	if err != nil {
		t.Fatal(err)
	}

	val := secret.Data["value"].(string)
	if val != "bar" {
		t.Fatalf("expected secret/foo.value to be bar, got %s", val)
	}
}

// Tests that passing a complete CoreConfig (sans plugins) results in a test
// Vault server with all of the listed backends present
func TestTestServerWithConf(t *testing.T) {
	conf := &vaulttest.CoreConfig{
		LogicalBackends: map[string]logical.Factory{
			"aws":       aws.Factory,
			"cassandra": cassandra.Factory,
			"consul":    consul.Factory,
			"database":  database.Factory,
			"nomad":     nomad.Factory,
			"pki":       pki.Factory,
			"rabbitmq":  rabbitmq.Factory,
			"ssh":       ssh.Factory,
			"totp":      totp.Factory,
			"transit":   transit.Factory,
		},
		CredentialBackends: map[string]logical.Factory{
			"approle":  approle.Factory,
			"aws":      aws.Factory,
			"cert":     cert.Factory,
			"github":   github.Factory,
			"ldap":     ldap.Factory,
			"okta":     okta.Factory,
			"radius":   radius.Factory,
			"userpass": userpass.Factory,
		},
		AuditBackends: map[string]audit.Factory{
			"file":   file.Factory,
			"socket": socket.Factory,
			"syslog": syslog.Factory,
		},
	}

	_, closer := TestServer(t, conf)
	defer closer()
}
