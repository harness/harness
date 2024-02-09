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

import { useHistory } from 'react-router-dom'
import React, { useMemo, useState } from 'react'
import { Container, Layout, FlexExpander, DropDown, ButtonVariation, Button, SelectOption } from '@harnessio/uicore'
import { sortBy } from 'lodash-es'
import { getConfig, getUsingFetch } from 'services/config'
import { useStrings } from 'framework/strings'
import { CodeIcon, GitInfoProps, makeDiffRefs, PullRequestFilterOption } from 'utils/GitUtils'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import type { TypesPrincipalInfo } from 'services/code'
import { useAppContext } from 'AppContext'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { permissionProps } from 'utils/Utils'
import css from './PullRequestsContentHeader.module.scss'

interface PullRequestsContentHeaderProps extends Pick<GitInfoProps, 'repoMetadata'> {
  loading?: boolean
  activePullRequestFilterOption?: string
  activePullRequestAuthorFilterOption?: string
  onPullRequestFilterChanged: React.Dispatch<React.SetStateAction<string>>
  onPullRequestAuthorFilterChanged: (authorFilter: string) => void
  onSearchTermChanged: (searchTerm: string) => void
}

export function PullRequestsContentHeader({
  loading,
  onPullRequestFilterChanged,
  onPullRequestAuthorFilterChanged,
  onSearchTermChanged,
  activePullRequestFilterOption = PullRequestFilterOption.OPEN,
  activePullRequestAuthorFilterOption,
  repoMetadata
}: PullRequestsContentHeaderProps) {
  const history = useHistory()
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const [filterOption, setFilterOption] = useState(activePullRequestFilterOption)
  const [authorFilterOption, setAuthorFilterOption] = useState(activePullRequestAuthorFilterOption)
  const [searchTerm, setSearchTerm] = useState('')
  const [query, setQuery] = useState<string>('')
  const [loadingAuthors, setLoadingAuthors] = useState<boolean>(false)
  const space = useGetSpaceParam()
  const { standalone, routingId } = useAppContext()
  const { hooks } = useAppContext()
  const permPushResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY'
      },
      permissions: ['code_repo_push']
    },
    [space]
  )
  const items = useMemo(
    () => [
      { label: getString('open'), value: PullRequestFilterOption.OPEN },
      { label: getString('merged'), value: PullRequestFilterOption.MERGED },
      { label: getString('closed'), value: PullRequestFilterOption.CLOSED },
      // { label: getString('draft'), value: PullRequestFilterOption.DRAFT },
      // { label: getString('yours'), value: PullRequestFilterOption.YOURS },
      { label: getString('all'), value: PullRequestFilterOption.ALL }
    ],
    [getString]
  )

  const bearerToken = hooks?.useGetToken?.() || ''
  const getAuthorsPromise = (): Promise<SelectOption[]> => {
    return new Promise((resolve, reject) => {
      setLoadingAuthors(true)
      try {
        getUsingFetch(getConfig('code/api/v1'), `/principals`, bearerToken, {
          queryParams: {
            query: query?.trim(),
            type: 'user',
            accountIdentifier: routingId
          }
        })
          .then((obj: TypesPrincipalInfo[]) => {
            const updatedAuthorsList = Array.isArray(obj)
              ? ([
                  ...(obj || []).map(item => ({
                    label: String(item?.display_name),
                    value: String(item?.id)
                  }))
                ] as SelectOption[])
              : ([] as SelectOption[])
            setLoadingAuthors(false)
            resolve(sortBy(updatedAuthorsList, item => item.label.toLowerCase()))
          })
          .catch(error => {
            setLoadingAuthors(false)
            reject(error)
          })
      } catch (error) {
        setLoadingAuthors(false)
        reject(error)
      }
    })
  }

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
        <FlexExpander />
        <DropDown
          value={authorFilterOption}
          items={() => getAuthorsPromise()}
          disabled={loadingAuthors}
          onChange={({ value, label }) => {
            setAuthorFilterOption(label as string)
            onPullRequestAuthorFilterChanged(value as string)
          }}
          popoverClassName={css.branchDropdown}
          icon="nav-user-profile"
          iconProps={{ size: 16 }}
          placeholder="Select Authors"
          addClearBtn={true}
          resetOnClose
          resetOnSelect
          resetOnQuery
          query={query}
          onQueryChange={newQuery => {
            setQuery(newQuery)
          }}
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
