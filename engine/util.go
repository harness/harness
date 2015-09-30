package engine

import (
	"encoding/json"
)

func encodeToLegacyFormat(t *Task) ([]byte, error) {
	// t.System.Plugins = append(t.System.Plugins, "plugins/*")

	// s := map[string]interface{}{}
	// s["repo"] = t.Repo
	// s["config"] = t.Config
	// s["secret"] = t.Secret
	// s["job"] = t.Job
	// s["system"] = t.System
	// s["workspace"] = map[string]interface{}{
	// 	"netrc": t.Netrc,
	// 	"keys":  t.Keys,
	// }
	// s["build"] = map[string]interface{}{
	// 	"number": t.Build.Number,
	// 	"status": t.Build.Status,
	// 	"head_commit": map[string]interface{}{
	// 		"sha":     t.Build.Commit,
	// 		"ref":     t.Build.Ref,
	// 		"branch":  t.Build.Branch,
	// 		"message": t.Build.Message,
	// 		"author": map[string]interface{}{
	// 			"login": t.Build.Author,
	// 			"email": t.Build.Email,
	// 		},
	// 	},
	// }
	return json.Marshal(t)
}
