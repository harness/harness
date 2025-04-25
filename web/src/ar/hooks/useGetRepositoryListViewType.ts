/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { RepositoryListViewTypeEnum } from '@ar/contexts/AppStoreContext'

import { useAppStore } from './useAppStore'
import { useFeatureFlags } from './useFeatureFlag'

export function useGetRepositoryListViewType() {
  const { repositoryListViewType } = useAppStore()
  const { HAR_TREE_VIEW_ENABLED } = useFeatureFlags()
  if (!HAR_TREE_VIEW_ENABLED) {
    return RepositoryListViewTypeEnum.LIST
  }
  return repositoryListViewType || RepositoryListViewTypeEnum.LIST
}
