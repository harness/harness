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
import type { FirewallExceptionStatusV3 } from '@harnessio/react-har-service-client'

import { useParentHooks } from '@ar/hooks'
import { DEFAULT_EXEMPTION_LIST_TABLE_SORT, DEFAULT_PAGE_INDEX, DEFAULT_PAGE_SIZE } from '@ar/constants'
import { type RepositoryPackageType, RepositoryScopeType } from '@ar/common/types'
import type { UseQueryParamsOptions } from '@ar/__mocks__/hooks'

export type ExemptionListPageQueryParams = {
  page: number
  size: number
  sort: string[]
  repositoryIds: string[]
  packageTypes: RepositoryPackageType[]
  status?: FirewallExceptionStatusV3
  searchTerm?: string
  scope: RepositoryScopeType
}

export const useExemptionListQueryParamOptions = (): UseQueryParamsOptions<ExemptionListPageQueryParams> => {
  const { useQueryParamsOptions } = useParentHooks()
  const _options = useQueryParamsOptions(
    {
      page: DEFAULT_PAGE_INDEX,
      size: DEFAULT_PAGE_SIZE,
      packageTypes: [],
      repositoryIds: [],
      sort: DEFAULT_EXEMPTION_LIST_TABLE_SORT,
      scope: RepositoryScopeType.NONE
    },
    { ignoreEmptyString: true }
  )
  const options = useMemo(() => ({ ..._options, strictNullHandling: true }), [_options])

  return options
}
