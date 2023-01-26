// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

import (
	"time"
)

// Config stores the system configuration.
type Config struct {
	// InstanceID specifis the ID of the gitness instance.
	// NOTE: If the value is not provided the hostname of the machine is used.
	InstanceID string `envconfig:"GITNESS_INSTANCE_ID"`
	Debug      bool   `envconfig:"GITNESS_DEBUG"`
	Trace      bool   `envconfig:"GITNESS_TRACE"`

	// URL defines the URLs via which the different parts of the service are reachable by.
	URL struct {
		// Git defines the external URL via which the GIT API is reachable.
		// NOTE: for routing to work properly, the request path & hostname reaching gitness
		// have to statisfy at least one of the following two conditions:
		// - Path ends with `/git`
		// - Hostname matches Config.Server.HTTP.GitHost
		// (this could be after proxy path / header rewrite).
		Git string `envconfig:"GITNESS_URL_GIT" default:"http://localhost:3000/git"`

		// API defines the external URL via which the rest API is reachable.
		// NOTE: for routing to work properly, the request path reaching gitness has to end with `/api`
		// (this could be after proxy path rewrite).
		API string `envconfig:"GITNESS_URL_API" default:"http://localhost:3000/api"`

		// APIInternal defines the internal URL via which the rest API is reachable.
		// NOTE: for routing to work properly, the request path reaching gitness has to end with `/api`
		// (this could be after proxy path rewrite).
		APIInternal string `envconfig:"GITNESS_URL_API_INTERNAL" default:"http://localhost:3000/api"`
	}

	// Git defines the git configuration parameters
	Git struct {
		Root           string `envconfig:"GITNESS_GIT_ROOT"`
		TmpDir         string `envconfig:"GITNESS_GIT_TMP_DIR"`          // directory for temporary data (repo clone)
		ServerHookPath string `envconfig:"GITNESS_GIT_SERVER_HOOK_PATH"` // path to binary used as git server hook
		DefaultBranch  string `envconfig:"GITNESS_GIT_DEFAULTBRANCH" default:"main"`
	}

	// Server defines the server configuration parameters.
	Server struct {
		// HTTP defines the http configuration parameters
		HTTP struct {
			Bind  string `envconfig:"GITNESS_HTTP_BIND" default:":3000"`
			Proto string `envconfig:"GITNESS_HTTP_PROTO"`
			Host  string `envconfig:"GITNESS_HTTP_HOST"`
			// GitHost is the host used to identify git traffic (OPTIONAL).
			GitHost string `envconfig:"GITNESS_HTTP_GIT_HOST" default:"git.localhost"`
		}

		// GRPC defines the grpc configuration parameters
		GRPC struct {
			Bind string `envconfig:"GITNESS_GRPC_BIND" default:":3001"`
		}

		// Acme defines Acme configuration parameters.
		Acme struct {
			Enabled bool   `envconfig:"GITNESS_ACME_ENABLED"`
			Endpont string `envconfig:"GITNESS_ACME_ENDPOINT"`
			Email   bool   `envconfig:"GITNESS_ACME_EMAIL"`
		}
	}

	// Database defines the database configuration parameters.
	Database struct {
		Driver     string `envconfig:"GITNESS_DATABASE_DRIVER" default:"sqlite3"`
		Datasource string `envconfig:"GITNESS_DATABASE_DATASOURCE" default:"database.sqlite3"`
	}

	// Token defines token configuration parameters.
	Token struct {
		Expire time.Duration `envconfig:"GITNESS_TOKEN_EXPIRE" default:"720h"`
	}

	// Cors defines http cors parameters
	Cors struct {
		AllowedOrigins   []string `envconfig:"GITNESS_CORS_ALLOWED_ORIGINS"   default:"*"`
		AllowedMethods   []string `envconfig:"GITNESS_CORS_ALLOWED_METHODS"   default:"GET,POST,PATCH,PUT,DELETE,OPTIONS"`
		AllowedHeaders   []string `envconfig:"GITNESS_CORS_ALLOWED_HEADERS"   default:"Origin,Accept,Accept-Language,Authorization,Content-Type,Content-Language,X-Requested-With,X-Request-Id"` //nolint:lll // struct tags can't be multiline
		ExposedHeaders   []string `envconfig:"GITNESS_CORS_EXPOSED_HEADERS"   default:"Link"`
		AllowCredentials bool     `envconfig:"GITNESS_CORS_ALLOW_CREDENTIALS" default:"true"`
		MaxAge           int      `envconfig:"GITNESS_CORS_MAX_AGE"           default:"300"`
	}

	// Secure defines http security parameters.
	Secure struct {
		AllowedHosts          []string          `envconfig:"GITNESS_HTTP_ALLOWED_HOSTS"`
		HostsProxyHeaders     []string          `envconfig:"GITNESS_HTTP_PROXY_HEADERS"`
		SSLRedirect           bool              `envconfig:"GITNESS_HTTP_SSL_REDIRECT"`
		SSLTemporaryRedirect  bool              `envconfig:"GITNESS_HTTP_SSL_TEMPORARY_REDIRECT"`
		SSLHost               string            `envconfig:"GITNESS_HTTP_SSL_HOST"`
		SSLProxyHeaders       map[string]string `envconfig:"GITNESS_HTTP_SSL_PROXY_HEADERS"`
		STSSeconds            int64             `envconfig:"GITNESS_HTTP_STS_SECONDS"`
		STSIncludeSubdomains  bool              `envconfig:"GITNESS_HTTP_STS_INCLUDE_SUBDOMAINS"`
		STSPreload            bool              `envconfig:"GITNESS_HTTP_STS_PRELOAD"`
		ForceSTSHeader        bool              `envconfig:"GITNESS_HTTP_STS_FORCE_HEADER"`
		BrowserXSSFilter      bool              `envconfig:"GITNESS_HTTP_BROWSER_XSS_FILTER"    default:"true"`
		FrameDeny             bool              `envconfig:"GITNESS_HTTP_FRAME_DENY"            default:"true"`
		ContentTypeNosniff    bool              `envconfig:"GITNESS_HTTP_CONTENT_TYPE_NO_SNIFF"`
		ContentSecurityPolicy string            `envconfig:"GITNESS_HTTP_CONTENT_SECURITY_POLICY"`
		ReferrerPolicy        string            `envconfig:"GITNESS_HTTP_REFERRER_POLICY"`
	}

	// Admin defines admin user params (no admin setup if either is empty)
	Admin struct {
		DisplayName string `envconfig:"GITNESS_ADMIN_DISPLAY_NAME"`
		Email       string `envconfig:"GITNESS_ADMIN_EMAIL"`
		Password    string `envconfig:"GITNESS_ADMIN_PASSWORD"`
	}

	Redis struct {
		Endpoint           string `envconfig:"GITNESS_REDIS_ENDPOINT"             default:"localhost:6379"`
		MaxRetries         int    `envconfig:"GITNESS_REDIS_MAX_RETRIES"          default:"3"`
		MinIdleConnections int    `envconfig:"GITNESS_REDIS_MIN_IDLE_CONNECTIONS" default:"0"`
		Password           string `envconfig:"GITNESS_REDIS_PASSWORD"`
	}

	Events struct {
		Mode                  string `envconfig:"GITNESS_EVENTS_MODE"                     default:"inmemory"`
		Namespace             string `envconfig:"GITNESS_EVENTS_NAMESPACE"                default:"gitness"`
		MaxStreamLength       int64  `envconfig:"GITNESS_EVENTS_MAX_STREAM_LENGTH"        default:"1000"`
		ApproxMaxStreamLength bool   `envconfig:"GITNESS_EVENTS_APPROX_MAX_STREAM_LENGTH" default:"true"`
	}

	Lock struct {
		// Provider is a name of distributed lock service like redis, memory, file etc...
		Provider      string        `envconfig:"GITNESS_LOCK_PROVIDER"          default:"inmemory"`
		Expiry        time.Duration `envconfig:"GITNESS_LOCK_EXPIRE"            default:"8s"`
		Tries         int           `envconfig:"GITNESS_LOCK_TRIES"             default:"32"`
		RetryDelay    time.Duration `envconfig:"GITNESS_LOCK_RETRY_DELAY"       default:"250ms"`
		DriftFactor   float64       `envconfig:"GITNESS_LOCK_DRIFT_FACTOR"      default:"0.01"`
		TimeoutFactor float64       `envconfig:"GITNESS_LOCK_TIMEOUT_FACTOR"    default:"0.05"`
		// AppNamespace is just service app prefix to avoid conflicts on key definition
		AppNamespace string `envconfig:"GITNESS_LOCK_APP_NAMESPACE"     default:"gitness"`
		// DefaultNamespace is when mutex doesn't specify custom namespace for their keys
		DefaultNamespace string `envconfig:"GITNESS_LOCK_DEFAULT_NAMESPACE" default:"default"`
	}

	Webhook struct {
		MaxRetryCount       int64 `envconfig:"GITNESS_WEBHOOK_MAX_RETRY_COUNT" default:"3"`
		Concurrency         int   `envconfig:"GITNESS_WEBHOOK_CONCURRENCY" default:"4"`
		AllowLoopback       bool  `envconfig:"GITNESS_WEBHOOK_ALLOW_LOOPBACK" default:"false"`
		AllowPrivateNetwork bool  `envconfig:"GITNESS_WEBHOOK_ALLOW_PRIVATE_NETWORK" default:"false"`
	}
}
