package yaml

import (
	"crypto/md5"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/drone/drone/model"
	yaml "gopkg.in/yaml.v2"
	"strings"
)

func GenScript(repo *model.Repo, build *model.Build, raw []byte, registry string) ([]byte, error) {
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
	publishPlugin := formPublish(repo, build, registry)
	config["clone"] = clonePlugin
	config["publish"] = publishPlugin

	script, err := yaml.Marshal(config)
	if err != nil {
		log.Errorln("can't marshel script", string(script), err)
		return nil, err
	}
	log.Infoln("gen script", string(script))

	return script, nil
}

func formPublish(repo *model.Repo, build *model.Build, registry string) interface{} {
	if len(registry) > 0 && !strings.HasSuffix(registry, "/") {
		registry = registry + "/"
	}
	refName := formRefName(build)

	docker := map[string]interface{}{
		"username": repo.Owner,
		"password": md5.Sum([]byte(repo.Owner)),
		"email":    fmt.Sprintf("%s@sryci.io", repo.Owner),
		"repo":     fmt.Sprintf("%s%s/%s", registry, repo.Owner, repo.Name),
		"tag": []string{
			"latest",
			fmt.Sprintf("%s_%s_%d", refName, build.Commit[:7], build.Number),
		},
	}

	return map[string]interface{}{"docker": docker}
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
		"recursive":               true,
		"skip_verify":             true,
		"submodule_update_remote": true,
	}
	return clone
}
