package yaml

import (
	"crypto/md5"
	"errors"
	"fmt"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/drone/drone/model"
	yaml "gopkg.in/yaml.v2"
)

var (
	//ErrNoBuild bad build section in yml
	ErrBadBuild = errors.New("bad build in yml")
)

func GenScript(repo *model.Repo, build *model.Build, raw []byte, insecure bool, registry, storage, pluginPrefix string) ([]byte, error) {
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
	buildPlugin, ok := config["build"]
	if !ok {
		return nil, ErrBadBuild
	}
	config["clone"] = formClone(repo, registry, pluginPrefix)
	config["publish"] = formPublish(repo, build, insecure, registry, storage, pluginPrefix)
	config["build"] = enhanceBuild(buildPlugin.(map[interface{}]interface{}))
	config["cache"] = formCache(repo, registry, pluginPrefix)

	log.Infof("new config\n %q", config)
	script, err := yaml.Marshal(config)
	if err != nil {
		log.Errorln("can't marshel script", string(script), err)
		return nil, err
	}
	log.Infoln("gen script", string(script))

	return script, nil
}

func formCache(repo *model.Repo, registry, pluginPrefix string) map[interface{}]interface{} {
	registryPrefix := registryPrefix(registry)
	imageCache := imageCache(repo)
	cache := map[interface{}]interface{}{
		"image":       fmt.Sprintf("%s%s%s", registryPrefix, pluginPrefix, "drone-cache"),
		"privileged":  true,
		"compression": "bzip2",
		"mount": []string{
			".git",
			imageCache,
		},
	}
	return cache
}

func registryPrefix(registry string) string {
	registryPrefix := registry
	if len(registryPrefix) > 0 && !strings.HasSuffix(registryPrefix, "/") {
		registryPrefix = registryPrefix + "/"
	}
	return registryPrefix
}

func enhanceBuild(build map[interface{}]interface{}) map[interface{}]interface{} {
	build["extra_hosts"] = getExtraHosts()
	build["privileged"] = true
	build["net"] = "bridge"
	return build
}

func getExtraHosts() []string {
	values := []string{}
	hostsVal := os.Getenv("DOCKER_EXTRA_HOSTS")
	if len(hostsVal) < 1 {
		return values
	}
	slices := strings.Split(strings.TrimSpace(hostsVal), " ")
	for _, slice := range slices {
		slice = strings.Replace(slice, " ", "", -1)
		if len(slice) > 0 {
			values = append(values, slice)
		}
	}
	return values
}

func formPublish(repo *model.Repo, build *model.Build, insecure bool, registry, storage, pluginPrefix string) interface{} {
	registryPrefix := registryPrefix(registry)
	refName := formRefName(build)
	cacheFile := imageCache(repo)
	salt := fmt.Sprintf("%x", md5.Sum([]byte(repo.Owner)))[:4]
	docker := map[string]interface{}{
		"image":          fmt.Sprintf("%s%s%s", registryPrefix, pluginPrefix, "drone-docker"),
		"username":       repo.Owner,
		"password":       fmt.Sprintf("%s%s", "Sryci1", salt),
		"email":          fmt.Sprintf("%s@sryci.io", repo.Owner),
		"privileged":     true,
		"insecure":       insecure,
		"registry":       registry,
		"storage_driver": storage,
		"repo":           fmt.Sprintf("%s%s/%s", registryPrefix, repo.Owner, repo.Name),
		"tag": []string{
			"latest",
			fmt.Sprintf("%s_%s_$$BUILD_NUMBER", refName, build.Commit[:7]),
		},
		"net":         "bridge",
		"extra_hosts": getExtraHosts(),
		"load":        cacheFile,
		"save": map[string]interface{}{
			"destination": cacheFile,
			"tag":         "latest",
		},
	}

	return map[string]interface{}{
		"docker": docker,
	}
}

func imageCache(repo *model.Repo) string {
	return fmt.Sprintf("%s/%s.tar", ".sryun", repo.Name)
}

func formRefName(build *model.Build) string {
	if build.Ref == "HEAD" {
		return "HEAD"
	} else {
		slices := strings.Split(build.Ref, "/")
		return slices[2]
	}
}

func formClone(repo *model.Repo, registry, pluginPrefix string) map[string]interface{} {
	registryPrefix := registryPrefix(registry)
	clone := map[string]interface{}{
		"image":                   fmt.Sprintf("%s%s%s", registryPrefix, pluginPrefix, "drone-git"),
		"privileged":              true,
		"pull":                    false,
		"recursive":               true,
		"skip_verify":             true,
		"submodule_update_remote": true,
		"net":         "bridge",
		"extra_hosts": getExtraHosts(),
	}
	return clone
}
