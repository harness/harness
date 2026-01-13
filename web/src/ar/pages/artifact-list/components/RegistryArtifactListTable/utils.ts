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

import { useMemo } from 'react'

import { useParentHooks } from '@ar/hooks'
import type { RepositoryPackageType } from '@ar/common/types'
import {
  DEFAULT_PACKAGE_LIST_TABLE_SORT,
  DEFAULT_PAGE_INDEX,
  DEFAULT_PAGE_SIZE,
  SoftDeleteFilterEnum
} from '@ar/constants'
import type { UseQueryParamsOptions } from '@ar/__mocks__/hooks'

export type RegistryArtifactListPageQueryParams = {
  page: number
  size: number
  sort: string[]
  searchTerm?: string
  artifactSearchTerm?: string
  isDeployedArtifacts: boolean
  packageTypes: RepositoryPackageType[]
  repositoryKey?: string
  labels: string[]
  softDeleteFilter?: SoftDeleteFilterEnum
}

export const useRegistryArtifactListQueryParamOptions =
  (): UseQueryParamsOptions<RegistryArtifactListPageQueryParams> => {
    const { useQueryParamsOptions } = useParentHooks()
    const _options = useQueryParamsOptions(
      {
        page: DEFAULT_PAGE_INDEX,
        size: DEFAULT_PAGE_SIZE,
        sort: DEFAULT_PACKAGE_LIST_TABLE_SORT,
        isDeployedArtifacts: false,
        packageTypes: [],
        labels: [],
        softDeleteFilter: SoftDeleteFilterEnum.EXCLUDE
      },
      { ignoreEmptyString: false }
    )
    const options = useMemo(() => ({ ..._options, strictNullHandling: true }), [_options])

    return options
  }
