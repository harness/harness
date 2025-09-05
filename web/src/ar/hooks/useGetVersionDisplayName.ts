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

import { Parent, RepositoryPackageType } from '@ar/common/types'
import { getShortDigest } from '@ar/pages/digest-list/utils'

import { useAppStore } from './useAppStore'
import { useFeatureFlags } from './useFeatureFlag'

export function useGetVersionDisplayName(packageType: RepositoryPackageType, version: string) {
  const { HAR_ENABLE_UNTAGGED_IMAGES_SUPPORT } = useFeatureFlags()
  const { parent } = useAppStore()

  switch (packageType) {
    case RepositoryPackageType.DOCKER:
    case RepositoryPackageType.HELM:
      if (HAR_ENABLE_UNTAGGED_IMAGES_SUPPORT || parent === Parent.OSS) {
        return getShortDigest(version)
      }
      return version
    default:
      return version
  }
}
