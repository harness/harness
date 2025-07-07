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
import { useHistory } from 'react-router-dom'
import { useGet } from 'restful-react'
import { Container, Layout, FlexExpander, DropDown, ButtonVariation, Button } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import { CodeIcon, GitInfoProps, makeDiffRefs, PullRequestFilterOption } from 'utils/GitUtils'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import type { TypesBranchTable, TypesPrincipalInfo } from 'services/code'
import { useAppContext } from 'AppContext'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { LabelFilterObj, PageBrowserProps, ScopeEnum, permissionProps } from 'utils/Utils'
import { useQueryParams } from 'hooks/useQueryParams'
import { LabelFilter } from 'components/Label/LabelFilter/LabelFilter'
import { PRBanner } from 'components/PRBanner/PRBanner'
import { PRAuthorFilter } from './PRAuthorFilter'
import css from './PullRequestsContentHeader.module.scss'

interface PullRequestsContentHeaderProps extends Pick<GitInfoProps, 'repoMetadata'> {
  loading?: boolean
  activePullRequestFilterOption?: string
  activePullRequestAuthorFilterOption?: string
  activePullRequestAuthorObj?: TypesPrincipalInfo | null
  activePullRequestLabelFilterOption?: LabelFilterObj[]
  onPullRequestFilterChanged: React.Dispatch<React.SetStateAction<string>>
  onPullRequestAuthorFilterChanged: (authorFilter: string) => void
  onPullRequestLabelFilterChanged: (labelFilter: LabelFilterObj[]) => void
  onSearchTermChanged: (searchTerm: string) => void
}

export function PullRequestsContentHeader({
  loading,
  onPullRequestFilterChanged,
  onPullRequestAuthorFilterChanged,
  onPullRequestLabelFilterChanged,
  onSearchTermChanged,
  activePullRequestFilterOption = PullRequestFilterOption.OPEN,
  activePullRequestAuthorFilterOption,
  activePullRequestAuthorObj,
  activePullRequestLabelFilterOption,
  repoMetadata
}: PullRequestsContentHeaderProps) {
  const history = useHistory()
  const { getString } = useStrings()
  const browserParams = useQueryParams<PageBrowserProps>()
  const [filterOption, setFilterOption] = useState(activePullRequestFilterOption)
  const [labelFilterOption, setLabelFilterOption] = useState(activePullRequestLabelFilterOption)
  const [searchTerm, setSearchTerm] = useState('')
  const space = useGetSpaceParam()
  const { hooks, standalone, routes } = useAppContext()
  const permPushResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY',
        resourceIdentifier: repoMetadata?.identifier as string
      },
      permissions: ['code_repo_push']
    },
    [space]
  )

  const { data: prCandidateBranches } = useGet<TypesBranchTable[]>({
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/candidates`
  })

  useEffect(() => {
    setLabelFilterOption(activePullRequestLabelFilterOption)
  }, [activePullRequestLabelFilterOption])

  useEffect(() => {
    setFilterOption(browserParams?.state as string)
  }, [browserParams])

  const items = useMemo(
    () => [
      { label: getString('open'), value: PullRequestFilterOption.OPEN },
      { label: getString('merged'), value: PullRequestFilterOption.MERGED },
      { label: getString('closed'), value: PullRequestFilterOption.CLOSED },
      { label: getString('all'), value: PullRequestFilterOption.ALL }
    ],
    [getString]
  )

  const bearerToken = hooks?.useGetToken?.() || ''

  return (
    <Container className={css.main} padding="xlarge">
      {prCandidateBranches?.map(branch => (
        <PRBanner key={branch.name} repoMetadata={repoMetadata} branch={branch} />
      ))}
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
        <FlexExpander />

        <LabelFilter
          labelFilterOption={labelFilterOption}
          setLabelFilterOption={setLabelFilterOption}
          onPullRequestLabelFilterChanged={onPullRequestLabelFilterChanged}
          bearerToken={bearerToken}
          repoMetadata={repoMetadata}
          spaceRef={space}
          filterScope={ScopeEnum.REPO_SCOPE}
        />

        <PRAuthorFilter
          onPullRequestAuthorFilterChanged={onPullRequestAuthorFilterChanged}
          activePullRequestAuthorFilterOption={activePullRequestAuthorFilterOption}
          activePullRequestAuthorObj={activePullRequestAuthorObj}
          bearerToken={bearerToken}
        />

        <DropDown
          value={filterOption}
          items={items}
          onChange={({ value }) => {
            setFilterOption(value as string)
            onPullRequestFilterChanged(value as string)
          }}
          popoverClassName={css.branchDropdown}
        />
        <Button
          variation={ButtonVariation.PRIMARY}
          text={getString('newPullRequest')}
          icon={CodeIcon.Add}
          onClick={() => {
            history.push(
              routes.toCODECompare({
                repoPath: repoMetadata?.path as string,
                diffRefs: makeDiffRefs(repoMetadata?.default_branch as string, '')
              })
            )
          }}
          {...permissionProps(permPushResult, standalone)}
        />
      </Layout.Horizontal>
    </Container>
  )
}
