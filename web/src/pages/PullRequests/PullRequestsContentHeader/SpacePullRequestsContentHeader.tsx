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

import React, { useEffect, useMemo, useState } from 'react'
import { Container, Layout, FlexExpander, DropDown, SelectOption } from '@harnessio/uicore'
import { Render } from 'react-jsx-match'
import { useStrings } from 'framework/strings'
import { PullRequestFilterOption, PullRequestReviewFilterOption, SpacePRTabs } from 'utils/GitUtils'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import type { TypesPrincipalInfo } from 'services/code'
import { useAppContext } from 'AppContext'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { ScopeLevelEnum, type LabelFilterObj, type PageBrowserProps, ScopeLevel } from 'utils/Utils'
import { useQueryParams } from 'hooks/useQueryParams'
import { LabelFilter } from 'components/Label/LabelFilter/LabelFilter'
import { PRAuthorFilter } from './PRAuthorFilter'
import css from './PullRequestsContentHeader.module.scss'

interface SpacePullRequestsContentHeaderProps {
  activeTab: SpacePRTabs
  loading?: boolean
  activePullRequestFilterOption?: string
  activePullRequestReviewFilterOption?: string
  activePullRequestAuthorFilterOption?: string
  activePullRequestAuthorObj?: TypesPrincipalInfo | null
  activePullRequestLabelFilterOption?: LabelFilterObj[]
  activePullRequestIncludeSubSpaceOption?: ScopeLevelEnum
  onPullRequestFilterChanged: React.Dispatch<React.SetStateAction<string>>
  onPullRequestReviewFilterChanged: React.Dispatch<React.SetStateAction<string>>
  onPullRequestAuthorFilterChanged: (authorFilter: string) => void
  onPullRequestLabelFilterChanged: (labelFilter: LabelFilterObj[]) => void
  onSearchTermChanged: (searchTerm: string) => void
  onPullRequestIncludeSubSpaceOptionChanged: React.Dispatch<React.SetStateAction<ScopeLevelEnum>>
}

export function SpacePullRequestsContentHeader({
  activeTab,
  loading,
  onPullRequestFilterChanged,
  onPullRequestReviewFilterChanged,
  onPullRequestAuthorFilterChanged,
  onPullRequestLabelFilterChanged,
  onPullRequestIncludeSubSpaceOptionChanged,
  onSearchTermChanged,
  activePullRequestFilterOption = PullRequestFilterOption.OPEN,
  activePullRequestReviewFilterOption,
  activePullRequestAuthorFilterOption,
  activePullRequestLabelFilterOption,
  activePullRequestAuthorObj,
  activePullRequestIncludeSubSpaceOption
}: SpacePullRequestsContentHeaderProps) {
  const { getString } = useStrings()
  const browserParams = useQueryParams<PageBrowserProps>()
  const [filterOption, setFilterOption] = useState(activePullRequestFilterOption)
  const [reviewFilterOption, setReviewFilterOption] = useState(activePullRequestReviewFilterOption)
  const [labelFilterOption, setLabelFilterOption] = useState(activePullRequestLabelFilterOption)
  const [searchTerm, setSearchTerm] = useState('')
  const space = useGetSpaceParam()
  const { hooks } = useAppContext()
  const [accountIdentifier, orgIdentifier, projectIdentifier] = space?.split('/') || []

  useEffect(() => {
    setLabelFilterOption(activePullRequestLabelFilterOption)
  }, [activePullRequestLabelFilterOption])

  useEffect(() => {
    setFilterOption(browserParams?.state as string)
  }, [browserParams, activeTab])

  useEffect(() => {
    activeTab === SpacePRTabs.REVIEW_REQUESTED && setReviewFilterOption(activePullRequestReviewFilterOption)
  }, [activePullRequestReviewFilterOption, activeTab])

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
    activePullRequestIncludeSubSpaceOption === ScopeLevelEnum.ALL
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

  const [scopeLabel, setScopeLabel] = useState<SelectOption>(currentScopeLabel ? currentScopeLabel : scopeOption[0])

  return (
    <Container className={css.main} padding="xlarge">
      <Layout.Horizontal spacing="medium">
        <SearchInputWithSpinner
          loading={loading}
          spinnerPosition="right"
          query={searchTerm}
          setQuery={value => {
            setSearchTerm(value)
            onSearchTermChanged(value)
          }}
        />
        <Render when={!projectIdentifier}>
          <DropDown
            placeholder={scopeLabel.label}
            value={scopeLabel}
            items={scopeOption}
            onChange={e => {
              onPullRequestIncludeSubSpaceOptionChanged(e.value as ScopeLevelEnum)
              setScopeLabel(e)
            }}
          />
        </Render>
        <FlexExpander />

        <LabelFilter
          labelFilterOption={labelFilterOption}
          setLabelFilterOption={setLabelFilterOption}
          onPullRequestLabelFilterChanged={onPullRequestLabelFilterChanged}
          bearerToken={bearerToken}
          filterScope={ScopeLevel.SPACE}
          spaceRef={space}
        />
        <Render when={activeTab === SpacePRTabs.REVIEW_REQUESTED}>
          <DropDown
            value={reviewFilterOption}
            items={reviewItems}
            onChange={({ value }) => {
              setReviewFilterOption(value as string)
              onPullRequestReviewFilterChanged(value as string)
            }}
            popoverClassName={css.branchDropdown}
          />
        </Render>
        <Render when={activeTab !== SpacePRTabs.CREATED}>
          <PRAuthorFilter
            onPullRequestAuthorFilterChanged={onPullRequestAuthorFilterChanged}
            activePullRequestAuthorFilterOption={activePullRequestAuthorFilterOption}
            activePullRequestAuthorObj={activePullRequestAuthorObj}
            bearerToken={bearerToken}
          />
        </Render>
        <DropDown
          value={filterOption}
          items={items}
          onChange={({ value }) => {
            setFilterOption(value as string)
            onPullRequestFilterChanged(value as string)
          }}
          popoverClassName={css.branchDropdown}
        />
      </Layout.Horizontal>
    </Container>
  )
}
