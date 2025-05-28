/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { useGet } from 'restful-react'
import type { TypesPullReqRepo } from 'services/code'
import { LIST_FETCHING_LIMIT, PageAction, ScopeLevelEnum } from 'utils/Utils'
import { DashboardFilter, PullRequestFilterOption } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
import { useGetSpaceParam } from './useGetSpaceParam'
import usePRFiltersContext from './usePRFiltersContext'

export const usePullRequestsData = (pageAction: { action: PageAction; timestamp: number }) => {
  const { currentUser } = useAppContext()
  const { state } = usePRFiltersContext()
  const {
    searchTerm,
    page,
    prStateFilter,
    includeSubspaces,
    reviewFilter,
    authorFilter,
    labelFilter,
    urlParams,
    encapFilter
  } = state
  const space = useGetSpaceParam()
  const [accountIdentifier, orgIdentifier, projectIdentifier] = space?.split('/') || []

  return useGet<TypesPullReqRepo[]>({
    path: `/api/v1/pullreq`,
    queryParams: {
      accountIdentifier,
      orgIdentifier,
      projectIdentifier,
      limit: String(LIST_FETCHING_LIMIT),
      exclude_description: true,
      page: page,
      sort: prStateFilter == PullRequestFilterOption.MERGED ? 'merged' : 'number',
      order: 'desc',
      query: searchTerm,
      include_subspaces: includeSubspaces === ScopeLevelEnum.ALL,
      state: urlParams.state ? urlParams.state : prStateFilter == PullRequestFilterOption.ALL ? '' : prStateFilter,
      ...((reviewFilter || encapFilter === DashboardFilter.REVIEW_REQUESTED) && {
        reviewer_id: Number(currentUser.id),
        review_decision: reviewFilter.split('&')
      }),
      ...(encapFilter === DashboardFilter.CREATED && { created_by: Number(currentUser.id) }),
      ...(authorFilter &&
        Number(authorFilter) !== 0 &&
        (encapFilter === DashboardFilter.REVIEW_REQUESTED || encapFilter === DashboardFilter.ALL) && {
          created_by: Number(authorFilter)
        }),

      ...(labelFilter.filter(({ type, valueId }) => type === 'label' || valueId === -1).length && {
        label_id: labelFilter
          .filter(({ type, valueId }) => type === 'label' || valueId === -1)
          .map(({ labelId }) => labelId)
      }),
      ...(labelFilter.filter(({ type }) => type === 'value').length && {
        value_id: labelFilter
          .filter(({ type, valueId }) => type === 'value' && valueId !== -1)
          .map(({ valueId }) => valueId)
      }),

      ...(page > 1
        ? pageAction?.action === PageAction.NEXT
          ? { updated_lt: pageAction.timestamp }
          : { updated_gt: pageAction.timestamp }
        : {})
    },
    queryParamStringifyOptions: {
      arrayFormat: 'repeat'
    },
    debounce: 500,
    lazy: !currentUser
  })
}
