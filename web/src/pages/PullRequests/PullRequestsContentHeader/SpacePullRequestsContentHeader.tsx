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

import React, { useCallback, useMemo } from 'react'
import {
  Container,
  Layout,
  FlexExpander,
  DropDown,
  SelectOption,
  MultiSelectDropDown,
  MultiSelectOption
} from '@harnessio/uicore'
import { Render } from 'react-jsx-match'
import { useStrings } from 'framework/strings'
import { DashboardFilter, PullRequestFilterOption, PullRequestReviewFilterOption } from 'utils/GitUtils'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import type { TypesPrincipalInfo } from 'services/code'
import { useAppContext } from 'AppContext'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { ScopeLevelEnum, ScopeLevel } from 'utils/Utils'
import { LabelFilter } from 'components/Label/LabelFilter/LabelFilter'
import usePRFiltersContext from 'hooks/usePRFiltersContext'
import ToggleTabsBtn from 'components/ToggleTabs/ToggleTabsBtn'
import { PRAuthorFilter } from './PRAuthorFilter'
import css from './PullRequestsContentHeader.module.scss'

interface SpacePullRequestsContentHeaderProps {
  loading?: boolean
  activePullRequestAuthorObj?: TypesPrincipalInfo | null
}

export function SpacePullRequestsContentHeader({
  loading,
  activePullRequestAuthorObj
}: SpacePullRequestsContentHeaderProps) {
  const { getString } = useStrings()

  const {
    state,
    setEncapFilter,
    setPrStateFilterOption,
    setSearchTerm,
    setIncludeSubspaces,
    setLabelFilter,
    setReviewFilter,
    setAuthorFilter
  } = usePRFiltersContext()

  const { searchTerm, prStateFilter, includeSubspaces, reviewFilter, authorFilter, labelFilter, encapFilter } = state

  const space = useGetSpaceParam()
  const { hooks } = useAppContext()
  const [accountIdentifier, orgIdentifier, projectIdentifier] = space?.split('/') || []

  const items = useMemo(
    () => [
      { label: getString('open'), value: PullRequestFilterOption.OPEN },
      { label: getString('merged'), value: PullRequestFilterOption.MERGED },
      { label: getString('closed'), value: PullRequestFilterOption.CLOSED },
      { label: getString('all'), value: PullRequestFilterOption.ALL }
    ],
    [getString]
  )

  const reviewItems = useMemo(
    () => [
      { label: getString('pending'), value: PullRequestReviewFilterOption.PENDING },
      { label: getString('approved'), value: PullRequestReviewFilterOption.APPROVED },
      { label: getString('pr.changesRequested'), value: PullRequestReviewFilterOption.CHANGES_REQUESTED }
    ],
    [getString]
  )

  const bearerToken = hooks?.useGetToken?.() || ''

  const scopeOption = [
    accountIdentifier && !orgIdentifier
      ? {
          label: getString('searchScope.allScopes'),
          value: ScopeLevelEnum.ALL
        }
      : null,
    accountIdentifier && !orgIdentifier
      ? { label: getString('searchScope.accOnly'), value: ScopeLevelEnum.CURRENT }
      : null,
    orgIdentifier ? { label: getString('searchScope.orgAndProj'), value: ScopeLevelEnum.ALL } : null,
    orgIdentifier ? { label: getString('searchScope.orgOnly'), value: ScopeLevelEnum.CURRENT } : null
  ].filter(Boolean) as SelectOption[]

  const currentScopeLabel =
    includeSubspaces === ScopeLevelEnum.ALL
      ? {
          label:
            accountIdentifier && !orgIdentifier
              ? getString('searchScope.allScopes')
              : getString('searchScope.orgAndProj'),
          value: ScopeLevelEnum.ALL
        }
      : {
          label:
            accountIdentifier && !orgIdentifier ? getString('searchScope.accOnly') : getString('searchScope.orgOnly'),
          value: ScopeLevelEnum.CURRENT
        }

  const dashboardEncapFilters = [
    { label: 'All', value: DashboardFilter.ALL },
    { label: 'Created', value: DashboardFilter.CREATED },
    { label: 'Review Requested', value: DashboardFilter.REVIEW_REQUESTED }
  ]

  const MemoizedPRAuthorFilter = useCallback(
    () => (
      <PRAuthorFilter
        onPullRequestAuthorFilterChanged={setAuthorFilter}
        activePullRequestAuthorFilterOption={authorFilter}
        activePullRequestAuthorObj={activePullRequestAuthorObj}
        bearerToken={bearerToken}
      />
    ),
    [authorFilter, encapFilter]
  )

  const modifiedReviewFilterOptions = (reviewOps: string | undefined) => {
    if (!reviewOps) {
      return [] as MultiSelectOption[]
    }
    return reviewOps?.split('&').map(revOps => {
      if (revOps === PullRequestReviewFilterOption.PENDING) {
        return { label: getString('pending'), value: PullRequestReviewFilterOption.PENDING }
      }
      if (revOps === PullRequestReviewFilterOption.APPROVED) {
        return { label: getString('approved'), value: PullRequestReviewFilterOption.APPROVED }
      }
      if (revOps === PullRequestReviewFilterOption.CHANGES_REQUESTED) {
        return { label: getString('pr.changesRequested'), value: PullRequestReviewFilterOption.CHANGES_REQUESTED }
      }
    }) as MultiSelectOption[]
  }

  return (
    <Container className={css.main} padding="xlarge">
      <Layout.Vertical flex={{ alignItems: 'flex-start' }} spacing="medium">
        <Layout.Horizontal spacing="medium">
          <ToggleTabsBtn
            currentTab={encapFilter}
            tabsList={dashboardEncapFilters}
            onTabChange={newTab => {
              setEncapFilter(newTab as DashboardFilter)
            }}
          />
        </Layout.Horizontal>
        <Layout.Horizontal spacing="medium" style={{ width: '100%' }}>
          <Render when={!projectIdentifier}>
            <DropDown
              placeholder={currentScopeLabel.label}
              value={currentScopeLabel}
              items={scopeOption}
              onChange={e => {
                setIncludeSubspaces(e.value as ScopeLevelEnum)
              }}
            />
          </Render>

          <MemoizedPRAuthorFilter />

          <DropDown
            value={prStateFilter}
            items={items}
            onChange={({ value }) => {
              setPrStateFilterOption(value as PullRequestFilterOption)
            }}
            popoverClassName={css.branchDropdown}
          />
          <LabelFilter
            labelFilterOption={labelFilter}
            onPullRequestLabelFilterChanged={setLabelFilter}
            bearerToken={bearerToken}
            filterScope={ScopeLevel.SPACE}
            spaceRef={space}
          />

          <MultiSelectDropDown
            value={modifiedReviewFilterOptions(reviewFilter)}
            items={reviewItems}
            resetOnSelect
            icon="time"
            iconProps={{ size: 16 }}
            placeholder={'Your Reviews'}
            onChange={option => {
              const optionString = option.length > 0 ? option.map(o => o.value).join('&') : ''
              setReviewFilter(optionString)
            }}
            popoverClassName={css.branchDropdown}
          />

          <FlexExpander />
          <SearchInputWithSpinner
            loading={loading}
            spinnerPosition="right"
            query={searchTerm}
            setQuery={setSearchTerm}
          />
        </Layout.Horizontal>
      </Layout.Vertical>
    </Container>
  )
}
