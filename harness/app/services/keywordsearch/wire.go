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

package keywordsearch

import (
	"context"

	gitevents "github.com/harness/gitness/app/events/git"
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideLocalIndexSearcher,
	ProvideIndexer,
	ProvideSearcher,
	ProvideService,
)

func ProvideService(ctx context.Context,
	config Config,
	gitReaderFactory *events.ReaderFactory[*gitevents.Reader],
	repoReaderFactory *events.ReaderFactory[*repoevents.Reader],
	repoStore store.RepoStore,
	indexer Indexer,
) (*Service, error) {
	return NewService(ctx,
		config,
		gitReaderFactory,
		repoReaderFactory,
		repoStore,
		indexer)
}

func ProvideLocalIndexSearcher() *LocalIndexSearcher {
	return NewLocalIndexSearcher()
}

func ProvideIndexer(l *LocalIndexSearcher) Indexer {
	return l
}

func ProvideSearcher(l *LocalIndexSearcher) Searcher {
	return l
}
