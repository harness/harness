/* Copyright 2024 Harness, Inc.
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

import { useGet } from 'restful-react'
import { useEffect, useRef, useMemo } from 'react'
import isEqual from 'lodash/isEqual'
import type { TypesGitspaceConfig } from 'cde-gitness/services'
import { useAppContext } from 'AppContext'
import type { EnumGitspaceSort } from 'services/cde'
import { getConfig } from 'services/config'
import { usePaginationProps } from './usePaginationProps'

interface GitspaceFilter {
  gitspace_owner?: string
  gitspace_states?: string[]
  query?: string
  org_identifiers?: string[]
  project_identifiers?: string[]
}

interface SortConfig {
  sort: EnumGitspaceSort
  order: 'asc' | 'desc'
}

interface PaginationInfo {
  totalItems: number
  totalPages: number
  gitspaceExists?: boolean
}

interface UsageDashboardPathParams {
  accountId: string
}

interface UsageDashboardQueryParams {
  routingId?: string
  page?: number
  limit?: number
  gitspace_owner?: string
  gitspace_states?: string[]
  order?: 'asc' | 'desc'
  sort?: EnumGitspaceSort
  query?: string
  org_identifiers?: string[]
  project_identifiers?: string[]
}

export const useUsageDashboardGitspaces = ({
  page,
  limit,
  filter,
  sortConfig
}: {
  page: number
  limit: number
  filter: GitspaceFilter
  sortConfig: SortConfig
}) => {
  const { accountInfo } = useAppContext()
  const accountId = accountInfo?.identifier || ''

  const getFilteredParams = (currentFilter: GitspaceFilter) => {
    const result: {
      project_identifiers?: string[]
      org_identifiers?: string[]
      gitspace_owner?: string
      gitspace_states?: string[]
      query?: string
    } = {
      ...currentFilter
    }

    if (currentFilter.project_identifiers && currentFilter.project_identifiers.length > 0) {
      const selectedOrgIds = currentFilter.project_identifiers
        .map(projectId => {
          const parts = projectId.split('/')
          return parts.length > 0 ? parts[0] : null
        })
        .filter(Boolean) as string[]

      if (currentFilter.org_identifiers && currentFilter.org_identifiers.length > 0) {
        const remainingOrgIds = currentFilter.org_identifiers.filter(orgId => !selectedOrgIds.includes(orgId))

        result.org_identifiers = remainingOrgIds.length > 0 ? remainingOrgIds : undefined
      } else {
        result.org_identifiers = undefined
      }
    }

    return result
  }

  const filteredParams = useMemo(
    () => getFilteredParams(filter),
    [filter.gitspace_owner, filter.query, filter.gitspace_states, filter.org_identifiers, filter.project_identifiers]
  )

  const gitspaceStatesKey = filter.gitspace_states?.join(',')
  const orgIdentifiersKey = filter.org_identifiers?.join(',')
  const projectIdentifiersKey = filter.project_identifiers?.join(',')

  const memoizedFilter = useMemo(
    () => filter,
    [filter.gitspace_owner, filter.query, gitspaceStatesKey, orgIdentifiersKey, projectIdentifiersKey]
  )

  const memoizedSortConfig = useMemo(() => sortConfig, [sortConfig.sort, sortConfig.order])

  const prevParamsRef = useRef({
    page,
    limit,
    filter: memoizedFilter,
    sortConfig: memoizedSortConfig
  })

  const { data, loading, error, refetch, response } = useGet<
    TypesGitspaceConfig[],
    any,
    UsageDashboardQueryParams,
    UsageDashboardPathParams
  >((pathParams: UsageDashboardPathParams) => `/accounts/${pathParams.accountId}/gitspaces`, {
    base: getConfig('cde/api/v1'),
    pathParams: { accountId },
    queryParams: {
      routingId: accountId,
      page, // Ensure page is correctly passed
      limit,
      gitspace_owner: filter.gitspace_owner || 'all',
      gitspace_states: filter.gitspace_states?.length ? filter.gitspace_states : undefined,
      order: sortConfig.order || 'desc',
      sort: sortConfig.sort || 'last_activated',
      query: filter.query || undefined,
      org_identifiers: filteredParams.org_identifiers,
      project_identifiers: filteredParams.project_identifiers
    },
    queryParamStringifyOptions: {
      arrayFormat: 'repeat'
    },
    debounce: 250,
    lazy: !accountId
  })

  useEffect(() => {
    if (accountId) {
      const currentParams = {
        page,
        limit,
        filter: memoizedFilter,
        sortConfig: memoizedSortConfig
      }

      const paramsChanged =
        currentParams.page !== prevParamsRef.current.page ||
        currentParams.limit !== prevParamsRef.current.limit ||
        !isEqual(currentParams.filter, prevParamsRef.current.filter) ||
        !isEqual(currentParams.sortConfig, prevParamsRef.current.sortConfig)

      if (paramsChanged) {
        const currentFilteredParams = getFilteredParams(memoizedFilter)

        refetch({
          pathParams: { accountId },
          queryParams: {
            routingId: accountId,
            page,
            limit,
            gitspace_owner: memoizedFilter.gitspace_owner || 'all',
            gitspace_states: memoizedFilter.gitspace_states?.length ? memoizedFilter.gitspace_states : undefined,
            order: memoizedSortConfig.order || 'desc',
            sort: memoizedSortConfig.sort || 'last_activated',
            query: memoizedFilter.query || undefined,
            org_identifiers: currentFilteredParams.org_identifiers,
            project_identifiers: currentFilteredParams.project_identifiers
          },
          queryParamStringifyOptions: {
            arrayFormat: 'repeat'
          }
        })
        prevParamsRef.current = currentParams
      }
    }
  }, [accountId, page, limit, memoizedFilter, memoizedSortConfig])

  const parsePaginationInfo = (): PaginationInfo | undefined => {
    if (!response) return undefined

    const totalItems = parseInt(response.headers.get('x-total') || '0')
    const totalPages = parseInt(response.headers.get('x-total-pages') || '0')
    const gitspaceExists = !!parseInt(response.headers.get('x-total-no-filter') || '0')

    return {
      totalItems,
      totalPages,
      gitspaceExists
    }
  }

  const paginationProps = usePaginationProps({
    pageIndex: page - 1,
    pageSize: limit,
    itemCount: parsePaginationInfo()?.totalItems || 0,
    pageCount: parsePaginationInfo()?.totalPages || 0
  })

  return {
    data: data || [],
    loading,
    error,
    refetch,
    pagination: parsePaginationInfo(),
    paginationProps
  }
}
