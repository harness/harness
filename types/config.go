// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

import "time"

// Config stores the system configuration.
type Config struct {
	Debug bool `envconfig:"APP_DEBUG"`
	Trace bool `envconfig:"APP_TRACE"`

	// Server defines the server configuration parameters.
	Server struct {
		Bind  string `envconfig:"APP_HTTP_BIND" default:":3000"`
		Proto string `envconfig:"APP_HTTP_PROTO"`
		Host  string `envconfig:"APP_HTTP_HOST"`

		// Acme defines Acme configuration parameters.
		Acme struct {
			Enabled bool   `envconfig:"APP_ACME_ENABLED"`
			Endpont string `envconfig:"APP_ACME_ENDPOINT"`
			Email   bool   `envconfig:"APP_ACME_EMAIL"`
		}
	}

	// Database defines the database configuration parameters.
	Database struct {
		Driver     string `envconfig:"APP_DATABASE_DRIVER" default:"sqlite3"`
		Datasource string `envconfig:"APP_DATABASE_DATASOURCE" default:"database.sqlite3"`
	}

	// Token defines token configuration parameters.
	Token struct {
		Expire time.Duration `envconfig:"APP_TOKEN_EXPIRE" default:"720h"`
	}

	// Cors defines http cors parameters
	Cors struct {
		AllowedOrigins   []string `envconfig:"APP_CORS_ALLOWED_ORIGINS"   default:"*"`
		AllowedMethods   []string `envconfig:"APP_CORS_ALLOWED_METHODS"   default:"GET,POST,PATCH,PUT,DELETE,OPTIONS"`
		AllowedHeaders   []string `envconfig:"APP_CORS_ALLOWED_HEADERS"   default:"Origin,Accept,Accept-Language,Authorization,Content-Type,Content-Language,X-Requested-With,X-Request-Id"` //nolint:lll // struct tags can't be multiline
		ExposedHeaders   []string `envconfig:"APP_CORS_EXPOSED_HEADERS"   default:"Link"`
		AllowCredentials bool     `envconfig:"APP_CORS_ALLOW_CREDENTIALS" default:"true"`
		MaxAge           int      `envconfig:"APP_CORS_MAX_AGE"           default:"300"`
	}

	// Secure defines http security parameters.
	Secure struct {
		AllowedHosts          []string          `envconfig:"APP_HTTP_ALLOWED_HOSTS"`
		HostsProxyHeaders     []string          `envconfig:"APP_HTTP_PROXY_HEADERS"`
		SSLRedirect           bool              `envconfig:"APP_HTTP_SSL_REDIRECT"`
		SSLTemporaryRedirect  bool              `envconfig:"APP_HTTP_SSL_TEMPORARY_REDIRECT"`
		SSLHost               string            `envconfig:"APP_HTTP_SSL_HOST"`
		SSLProxyHeaders       map[string]string `envconfig:"APP_HTTP_SSL_PROXY_HEADERS"`
		STSSeconds            int64             `envconfig:"APP_HTTP_STS_SECONDS"`
		STSIncludeSubdomains  bool              `envconfig:"APP_HTTP_STS_INCLUDE_SUBDOMAINS"`
		STSPreload            bool              `envconfig:"APP_HTTP_STS_PRELOAD"`
		ForceSTSHeader        bool              `envconfig:"APP_HTTP_STS_FORCE_HEADER"`
		BrowserXSSFilter      bool              `envconfig:"APP_HTTP_BROWSER_XSS_FILTER"    default:"true"`
		FrameDeny             bool              `envconfig:"APP_HTTP_FRAME_DENY"            default:"true"`
		ContentTypeNosniff    bool              `envconfig:"APP_HTTP_CONTENT_TYPE_NO_SNIFF"`
		ContentSecurityPolicy string            `envconfig:"APP_HTTP_CONTENT_SECURITY_POLICY"`
		ReferrerPolicy        string            `envconfig:"APP_HTTP_REFERRER_POLICY"`
	}
}
