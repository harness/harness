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

package languageanalyzer

import (
	"context"
	"fmt"

	gitevents "github.com/harness/gitness/app/events/git"
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideAnalyzer,
)

func ProvideAnalyzer(
	ctx context.Context,
	config *types.Config,
	repoEvReaderFactory *events.ReaderFactory[*repoevents.Reader],
	gitReaderFactory *events.ReaderFactory[*gitevents.Reader],
	tx dbtx.Transactor,
	repoStore store.RepoStore,
	repoFinder refcache.RepoFinder,
	repoLangStore store.RepoLangStore,
	gitService git.Interface,
) (LanguageAnalyzer, error) {
	a := NewAnalyzer(tx, repoStore, repoFinder, repoLangStore, gitService)
	err := RegisterEventListeners(
		ctx,
		config.InstanceID,
		repoEvReaderFactory,
		gitReaderFactory,
		a,
	)
	if err != nil {
		return Analyzer{}, fmt.Errorf(
			"failed to register language analyzer event listeners: %w", err,
		)
	}

	return a, nil
}
