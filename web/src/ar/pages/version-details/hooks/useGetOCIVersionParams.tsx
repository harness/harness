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

import { OCIVersionType } from '@ar/common/types'
import type { VersionDetailsPathParams } from '@ar/routes/types'
import { useDecodedParams, useFeatureFlags, useParentHooks } from '@ar/hooks'
import type { DockerVersionDetailsQueryParams } from '../DockerVersion/types'

export default function useGetOCIVersionParams() {
  const { useQueryParams } = useParentHooks()
  const { tag, digest } = useQueryParams<DockerVersionDetailsQueryParams>()
  const pathParams = useDecodedParams<VersionDetailsPathParams>()
  const { HAR_ENABLE_UNTAGGED_IMAGES_SUPPORT } = useFeatureFlags()

  const versionIdentifier = HAR_ENABLE_UNTAGGED_IMAGES_SUPPORT
    ? tag ?? pathParams.versionIdentifier
    : pathParams.versionIdentifier
  const versionType = HAR_ENABLE_UNTAGGED_IMAGES_SUPPORT
    ? tag
      ? OCIVersionType.TAG
      : OCIVersionType.DIGEST
    : OCIVersionType.TAG

  return { versionIdentifier, versionType, digest }
}
