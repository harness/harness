package yaml

import (
	"crypto/md5"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/drone/drone/model"
	yaml "gopkg.in/yaml.v2"
	"strings"
)

func GenScript(repo *model.Repo, build *model.Build, raw []byte, insecure bool, registry, storage string) ([]byte, error) {
	config := map[interface{}]interface{}{}
	err := yaml.Unmarshal(raw, &config)
	if err != nil {
		log.Errorln("can't parse script", err)
		return nil, err
	}
	log.Infof("config\n %q", config)

	delete(config, "cache")
	delete(config, "compose")
	delete(config, "deploy")
	delete(config, "notify")

	clonePlugin := formClone(repo)
	publishPlugin := formPublish(repo, build, insecure, registry, storage)
	config["clone"] = clonePlugin
	config["publish"] = publishPlugin

	log.Infof("new config\n %q", config)
	script, err := yaml.Marshal(config)
	if err != nil {
		log.Errorln("can't marshel script", string(script), err)
		return nil, err
	}
	log.Infoln("gen script", string(script))

	return script, nil
}

func formPublish(repo *model.Repo, build *model.Build, insecure bool, registry, storage string) interface{} {
	registryPrefix := registry
	if len(registryPrefix) > 0 && !strings.HasSuffix(registryPrefix, "/") {
		registryPrefix = registryPrefix + "/"
	}
	refName := formRefName(build)
	password := "Sryci1" + fmt.Sprintf("%x", md5.Sum([]byte(repo.Owner)))[:4]
	docker := map[string]interface{}{
		//"image": "testregistry.dataman.io/drone-plugins-docker",
		//"image": "10.3.10.36:5000/library/drone-docker:0.1",
		"image": "plugins/drone-docker",
		//"username": "admin",
		//"password": "admin",
		"username":       repo.Owner,
		"password":       password,
		"privileged":     true,
		"pull":           true,
		"insecure":       insecure,
		"registry":       registry,
		"storage_driver": storage,
		"email":          fmt.Sprintf("%s@sryci.io", repo.Owner),
		"repo":           fmt.Sprintf("%s%s/%s", registryPrefix, repo.Owner, repo.Name),
		//"repo": fmt.Sprintf("%s%s", registryPrefix, repo.Name),
		"tag": []string{
			"latest",
			fmt.Sprintf("%s_%s_$$BUILD_NUMBER", refName, build.Commit[:7]),
		},
	}

	return map[string]interface{}{
		"docker": docker,
	}
}

func formRefName(build *model.Build) string {
	if build.Ref == "HEAD" {
		return "HEAD"
	} else {
		slices := strings.Split(build.Ref, "/")
		return slices[2]
	}
}

func formClone(repo *model.Repo) map[string]interface{} {
	clone := map[string]interface{}{
		"image":                   "testregistry.dataman.io/drone-git",
		"privileged":              true,
		"pull":                    true,
		"recursive":               true,
		"skip_verify":             true,
		"submodule_update_remote": true,
	}
	return clone
}
