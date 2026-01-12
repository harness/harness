// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package langstats

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/harness/gitness/git/api"

	"github.com/rs/zerolog/log"
	"golang.org/x/exp/maps"
)

const Unclassified = "Unclassified"

type LangStat struct {
	Bytes int64
	Files int64
}

func AnalyzeRepoLanguages(
	ctx context.Context,
	nodes []api.TreeNode,
) map[string]*LangStat {
	stats := make(map[string]*LangStat)

	unclassifiedExtensions := map[string]struct{}{}

	for _, n := range nodes {
		// Skip directories, submodules, symlinks
		if n.IsDir() || n.IsSubmodule() || n.IsLink() {
			continue
		}

		// Extension-based guess only; uniqueness matters once content detection is added.
		lang, _ := GetLanguageByExtension(filepath.Ext(n.Path))

		if _, ok := stats[lang]; !ok {
			stats[lang] = &LangStat{}
		}

		stats[lang].Bytes += n.Size
		stats[lang].Files++

		if lang == Unclassified {
			unclassifiedExtensions[filepath.Ext(n.Path)] = struct{}{}
		}
	}

	log.Info().Ctx(ctx).
		Strs("unclassified_extensions", maps.Keys(unclassifiedExtensions)).
		Msgf("Detected %d unclassified file extensions during language analysis", len(unclassifiedExtensions))

	return stats
}

// GetLanguageByExtension returns the first matching language for an extension
// and whether the extension maps to a single language.
func GetLanguageByExtension(ext string) (lang string, unique bool) {
	if ext == "" {
		return "", false
	}

	langs, ok := progLangsByExt[strings.ToLower(ext)]
	if !ok || len(langs) == 0 {
		return Unclassified, false
	}

	return langs[0], len(langs) == 1
}
