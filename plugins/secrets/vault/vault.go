// Copyright 2018 Drone.IO Inc
// Use of this software is governed by the Drone Enterpise License
// that can be found in the LICENSE file.

package vault

import (
	"path"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/drone/drone/model"
	"github.com/drone/drone/plugins/secrets"

	"github.com/hashicorp/vault/api"
	"gopkg.in/yaml.v2"
)

// yaml configuration representation
//
//    secrets:
//      docker_username:
//        file: path/to/docker/username
//      docker_password:
//        file: path/to/docker/password
//
type vaultConfig struct {
	Secrets map[string]struct {
		Driver     string
		DriverOpts struct {
			Path string
			Key  string
		} `yaml:"driver_opts"`

		// deprecated. do not use.
		Vault string
		Path  string
		File  string
	}
}

type vault struct {
	store    model.ConfigStore
	client   *api.Client
	ttl      time.Duration
	renew    time.Duration
	auth     string
	kubeAuth kubeAuth
	done     chan struct{}
}

type kubeAuth struct {
	addr, role, mount string
}

// New returns a new store with secrets loaded from vault.
func New(store model.ConfigStore, opts ...Opts) (secrets.Plugin, error) {
	client, err := api.NewClient(nil)
	if err != nil {
		return nil, err
	}
	v := &vault{
		store:  store,
		client: client,
	}
	for _, opt := range opts {
		opt(v)
	}
	if v.auth == "kubernetes" {
		err = v.initKubernetes()
		if err != nil {
			return nil, err
		}
	}
	v.start() // start the refresh process.
	return v, nil
}

func (v *vault) initKubernetes() error {
	token, ttl, err := getKubernetesToken(
		v.kubeAuth.addr,
		v.kubeAuth.role,
		v.kubeAuth.mount,
		"/var/run/secrets/kubernetes.io/serviceaccount/token",
	)
	if err != nil {
		logrus.Debugf("vault: failed to obtain token via kubernetes-auth backend: %s", err)
		return err
	}

	v.client.SetToken(token)
	v.ttl = ttl
	v.renew = ttl / 2
	return nil
}

func (v *vault) SecretListBuild(repo *model.Repo, build *model.Build) ([]*model.Secret, error) {
	return v.list(repo, build)
}

func (v *vault) list(repo *model.Repo, build *model.Build) ([]*model.Secret, error) {
	conf, err := v.store.ConfigLoad(build.ConfigID)
	if err != nil {
		return nil, err
	}
	var (
		in  = []byte(conf.Data)
		out = new(vaultConfig)

		secrets []*model.Secret
	)
	err = yaml.Unmarshal(in, out)
	if err != nil {
		return nil, err
	}
	for key, val := range out.Secrets {
		var path, field string
		switch {
		case val.Path != "":
			path = val.Path
		case val.File != "":
			path = val.File
		case val.Vault != "":
			path = val.Vault
		case val.DriverOpts.Path != "":
			path = val.DriverOpts.Path
			field = val.DriverOpts.Key
		}
		if field == "" {
			field = "value"
		}

		if path == "" {
			continue
		}

		logrus.Debugf("vault: read secret: %s", path)

		vaultSecret, err := v.get(path, field)
		if err != nil {
			logrus.Debugf("vault: read secret failed: %s: %s", path, err)
			return nil, err
		}
		if vaultSecret == nil {
			logrus.Debugf("vault: read secret failed: %s: not found or empty value", path)
			continue
		}
		if !vaultSecret.Match(repo.FullName) {
			logrus.Debugf("vault: read secret: %s: restricted: %s", path, repo.FullName)
			continue
		}

		logrus.Debugf("vault: read secret success: %s", err)

		secrets = append(secrets, &model.Secret{
			Name:   key,
			Value:  vaultSecret.Value,
			Events: vaultSecret.Event,
			Images: vaultSecret.Image,
		})
	}
	return secrets, nil
}

func (v *vault) get(path, key string) (*vaultSecret, error) {
	secret, err := v.client.Logical().Read(path)
	if err != nil {
		return nil, err
	}
	if secret == nil || secret.Data == nil {
		return nil, nil
	}
	return parseVaultSecret(secret.Data, key), nil
}

// start starts the renewal loop.
func (v *vault) start() {
	if v.renew == 0 || v.ttl == 0 {
		logrus.Debugf("vault: token renewal disabled")
		return
	}
	if v.done != nil {
		close(v.done)
	}
	logrus.Infof("vault: token renewal enabled: renew every %v", v.renew)
	v.done = make(chan struct{})
	if v.renew != 0 {
		go v.renewLoop()
	}
}

// stop stops the renewal loop.
func (v *vault) stop() {
	close(v.done)
}

func (v *vault) renewLoop() {
	for {
		select {
		case <-time.After(v.renew):
			incr := int(v.ttl / time.Second)

			logrus.Debugf("vault: refreshing token: increment %v", v.ttl)
			_, err := v.client.Auth().Token().RenewSelf(incr)
			if err != nil {
				logrus.Errorf("vault: refreshing token failed: %s", err)
			} else {
				logrus.Debugf("vault: refreshing token succeeded")
			}
		case <-v.done:
			return
		}
	}
}

type vaultSecret struct {
	Value string
	Image []string
	Event []string
	Repo  []string
}

func parseVaultSecret(data map[string]interface{}, key string) *vaultSecret {
	secret := new(vaultSecret)

	if vvalue, ok := data[key]; ok {
		if svalue, ok := vvalue.(string); ok {
			secret.Value = svalue
		}
	}
	if vimage, ok := data["image"]; ok {
		if simage, ok := vimage.(string); ok {
			secret.Image = strings.Split(simage, ",")
		}
	}
	if vevent, ok := data["event"]; ok {
		if sevent, ok := vevent.(string); ok {
			secret.Event = strings.Split(sevent, ",")
		}
	}
	if vrepo, ok := data["repo"]; ok {
		if srepo, ok := vrepo.(string); ok {
			secret.Repo = strings.Split(srepo, ",")
		}
	}
	if secret.Event == nil {
		secret.Event = []string{}
	}
	if secret.Image == nil {
		secret.Image = []string{}
	}
	if secret.Repo == nil {
		secret.Repo = []string{}
	}
	return secret
}

func (v *vaultSecret) Match(name string) bool {
	if len(v.Repo) == 0 {
		return true
	}
	for _, pattern := range v.Repo {
		if ok, _ := path.Match(pattern, name); ok {
			return true
		}
	}
	return false
}
