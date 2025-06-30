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

import React, { useCallback, useEffect } from 'react'
import { isEmpty } from 'lodash-es'
import { useAppContext } from 'AppContext'
import { useQueryParams } from 'hooks/useQueryParams'
import { useUpdateQueryParams } from 'hooks/useUpdateQueryParams'
import { DashboardFilter, PullRequestFilterOption } from 'utils/GitUtils'
import { LabelFilterObj, PageBrowserProps, ScopeLevelEnum } from 'utils/Utils'

type FilterState = {
  searchTerm: string
  encapFilter: DashboardFilter
  includeSubspaces: ScopeLevelEnum
  prStateFilter: PullRequestFilterOption
  reviewFilter: string
  authorFilter?: string
  labelFilter: LabelFilterObj[]
  page: number
  urlParams: PageBrowserProps
}

enum PRFilterActionsEnum {
  SET_ENCAP_FILTER = 'SET_ENCAP_FILTER',
  SET_AUTHOR_FILTER = 'SET_AUTHOR_FILTER',
  SET_REVIEW_FILTER = 'SET_REVIEW_FILTER',
  SET_LABEL_FILTER = 'SET_LABEL_FILTER',
  SET_INCLUDE_SUBSPACES = 'SET_INCLUDE_SUBSPACES',
  SET_SEARCH_TERM = 'SET_SEARCH_TERM',
  SET_PR_STATE_FILTER_OPTION = 'SET_PR_STATE_FILTER_OPTION',
  SET_PAGE = 'SET_PAGE',
  SYNC_URL_PARAMS = 'SYNC_URL_PARAMS',
  RESET_FILTERS = 'RESET_FILTERS'
}

type PRFilterActions =
  | { type: PRFilterActionsEnum.SET_ENCAP_FILTER; payload: { filter: DashboardFilter; currentUserId: number } }
  | { type: PRFilterActionsEnum.SET_AUTHOR_FILTER; payload: { author: string; currentUserId: number } }
  | { type: PRFilterActionsEnum.SET_REVIEW_FILTER; payload: { reviewFilter: string; currentUserId: number } }
  | { type: PRFilterActionsEnum.SET_LABEL_FILTER; payload: LabelFilterObj[] }
  | { type: PRFilterActionsEnum.SET_INCLUDE_SUBSPACES; payload: ScopeLevelEnum }
  | { type: PRFilterActionsEnum.SET_SEARCH_TERM; payload: string }
  | { type: PRFilterActionsEnum.SET_PR_STATE_FILTER_OPTION; payload: PullRequestFilterOption }
  | { type: PRFilterActionsEnum.SET_PAGE; payload: number }
  | { type: PRFilterActionsEnum.SYNC_URL_PARAMS; payload: Record<string, string> }
  | { type: PRFilterActionsEnum.RESET_FILTERS; payload: FilterState }

const PRFilterReducer = (state: FilterState, action: PRFilterActions): FilterState => {
  switch (action.type) {
    case PRFilterActionsEnum.SET_ENCAP_FILTER:
      if (action.payload.filter === DashboardFilter.ALL) {
        return {
          ...state,
          searchTerm: '',
          encapFilter: action.payload.filter,
          reviewFilter: '',
          authorFilter: '',
          labelFilter: [],
          page: 1,
          urlParams: {}
        }
      } else if (action.payload.filter === DashboardFilter.REVIEW_REQUESTED) {
        return {
          ...state,
          encapFilter: action.payload.filter,
          reviewFilter: 'pending',
          authorFilter: '',
          page: 1,
          urlParams: {}
        }
      }

      return {
        ...state,
        encapFilter: action.payload.filter,
        authorFilter: String(action.payload.currentUserId),
        reviewFilter: '',
        page: 1
      }
    case PRFilterActionsEnum.SET_AUTHOR_FILTER: {
      const { author, currentUserId } = action.payload
      if (author === String(currentUserId) && state.encapFilter !== DashboardFilter.REVIEW_REQUESTED) {
        return {
          ...state,
          authorFilter: author,
          encapFilter: DashboardFilter.CREATED,
          page: 1
        }
      } else if (state.encapFilter === DashboardFilter.CREATED && author !== String(currentUserId)) {
        return {
          ...state,
          authorFilter: author,
          encapFilter: DashboardFilter.ALL,
          page: 1
        }
      }
      return {
        ...state,
        authorFilter: author,
        page: 1
      }
    }
    case PRFilterActionsEnum.SET_REVIEW_FILTER:
      if (action.payload.reviewFilter !== '') {
        const newState = {
          ...state,
          reviewFilter: action.payload.reviewFilter,
          encapFilter: DashboardFilter.REVIEW_REQUESTED,
          page: 1
        }
        if (state.authorFilter === String(action.payload.currentUserId)) {
          delete newState.authorFilter
        }
        return newState
      }
      return {
        ...state,
        reviewFilter: action.payload.reviewFilter,
        encapFilter: DashboardFilter.ALL,
        page: 1
      }
    case PRFilterActionsEnum.SET_LABEL_FILTER:
      return {
        ...state,
        labelFilter: action.payload,
        page: 1
      }
    case PRFilterActionsEnum.SET_INCLUDE_SUBSPACES:
      return {
        ...state,
        includeSubspaces: action.payload,
        page: 1
      }
    case PRFilterActionsEnum.SET_SEARCH_TERM:
      return {
        ...state,
        searchTerm: action.payload,
        page: 1
      }
    case PRFilterActionsEnum.SET_PR_STATE_FILTER_OPTION:
      return {
        ...state,
        prStateFilter: action.payload,
        page: 1
      }
    case PRFilterActionsEnum.SET_PAGE:
      return {
        ...state,
        page: action.payload
      }
    case PRFilterActionsEnum.SYNC_URL_PARAMS:
      return {
        ...state,
        urlParams: action.payload
      }
    case PRFilterActionsEnum.RESET_FILTERS:
      return { ...action.payload }
    default:
      return state
  }
}

type PRFilterContextType = {
  state: FilterState
  dispatch: (action: PRFilterActions) => void
  resetFilters: (initialState: FilterState) => void
  setEncapFilter: (filter: DashboardFilter) => void
  setAuthorFilter: (author: string) => void
  setReviewFilter: (reviewFilter: string) => void
  setLabelFilter: (labelFilter: LabelFilterObj[]) => void
  setIncludeSubspaces: (includeSubspaces: ScopeLevelEnum) => void
  setSearchTerm: (searchTerm: string) => void
  setPrStateFilterOption: (prStateFilterOption: PullRequestFilterOption) => void
  setPage: (page: number) => void
  syncUrlParams: (params: Record<string, string>) => void
}

export const PRFilterContext = React.createContext<PRFilterContextType | null>(null)

export const PRFilterProvider = ({ children }: { children: JSX.Element }): JSX.Element => {
  const browserParams = useQueryParams<PageBrowserProps>()
  const { currentUser } = useAppContext()
  // Initial state
  const initialState: FilterState = {
    searchTerm: '',
    encapFilter: browserParams?.review
      ? DashboardFilter.REVIEW_REQUESTED
      : browserParams.author === String(currentUser.id) || isEmpty(browserParams) //for fresh mount
      ? DashboardFilter.CREATED
      : DashboardFilter.ALL,
    includeSubspaces: browserParams?.recursive === 'true' ? ScopeLevelEnum.ALL : ScopeLevelEnum.CURRENT,
    prStateFilter: (browserParams?.state as PullRequestFilterOption) || (PullRequestFilterOption.OPEN as string),
    reviewFilter: (browserParams?.review as string) || '',
    authorFilter: isEmpty(browserParams) ? String(currentUser.id) : (browserParams?.author as string),
    labelFilter: [],
    page: browserParams.page ? parseInt(browserParams.page) : 1,
    urlParams: {}
  }

  const [state, dispatch] = React.useReducer(PRFilterReducer, initialState)

  const { replaceQueryParams } = useUpdateQueryParams<PageBrowserProps>()

  useEffect(() => {
    replaceQueryParams(
      {
        ...(!isEmpty(state.prStateFilter) && { state: state.prStateFilter }),
        ...(!isEmpty(state.reviewFilter) && { review: state.reviewFilter }),
        ...(!isEmpty(state.authorFilter) && { author: state.authorFilter }),
        ...(!isEmpty(state.includeSubspaces) && {
          recursive: state.includeSubspaces === ScopeLevelEnum.ALL ? 'true' : 'false'
        }),
        ...(state.page > 1 && { page: state.page.toString() })
      },
      undefined,
      true
    )
  }, [state])

  useEffect(() => {
    if (currentUser?.id && state.encapFilter === DashboardFilter.CREATED) {
      setAuthorFilter(String(currentUser.id))
    }
  }, [currentUser, state.encapFilter])

  const setEncapFilter = useCallback((filter: DashboardFilter) => {
    dispatch({ type: PRFilterActionsEnum.SET_ENCAP_FILTER, payload: { filter, currentUserId: currentUser?.id } })
  }, [])

  const setAuthorFilter = useCallback(
    (author: string) => {
      dispatch({ type: PRFilterActionsEnum.SET_AUTHOR_FILTER, payload: { author, currentUserId: currentUser?.id } })
    },
    [currentUser]
  )

  const setReviewFilter = useCallback((reviewFilter: string) => {
    dispatch({ type: PRFilterActionsEnum.SET_REVIEW_FILTER, payload: { reviewFilter, currentUserId: currentUser?.id } })
  }, [])

  const setLabelFilter = useCallback((labelFilter: LabelFilterObj[]) => {
    dispatch({ type: PRFilterActionsEnum.SET_LABEL_FILTER, payload: labelFilter })
  }, [])

  const setIncludeSubspaces = useCallback((includeSubspaces: ScopeLevelEnum) => {
    dispatch({ type: PRFilterActionsEnum.SET_INCLUDE_SUBSPACES, payload: includeSubspaces })
  }, [])

  const setSearchTerm = useCallback((searchTerm: string) => {
    dispatch({ type: PRFilterActionsEnum.SET_SEARCH_TERM, payload: searchTerm })
  }, [])

  const setPrStateFilterOption = useCallback((prStateFilterOption: PullRequestFilterOption) => {
    dispatch({ type: PRFilterActionsEnum.SET_PR_STATE_FILTER_OPTION, payload: prStateFilterOption })
  }, [])

  const setPage = useCallback((page: number) => {
    dispatch({ type: PRFilterActionsEnum.SET_PAGE, payload: page })
  }, [])

  const syncUrlParams = useCallback((params: Record<string, string>) => {
    dispatch({ type: PRFilterActionsEnum.SYNC_URL_PARAMS, payload: params })
  }, [])

  return (
    <PRFilterContext.Provider
      value={{
        state,
        dispatch,
        resetFilters: () => dispatch({ type: PRFilterActionsEnum.RESET_FILTERS, payload: initialState }),
        setEncapFilter,
        setAuthorFilter,
        setReviewFilter,
        setLabelFilter,
        setIncludeSubspaces,
        setSearchTerm,
        setPrStateFilterOption,
        setPage,
        syncUrlParams
      }}>
      {children}
    </PRFilterContext.Provider>
  )
}
