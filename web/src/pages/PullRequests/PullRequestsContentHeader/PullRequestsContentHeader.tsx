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
import React, { useEffect, useMemo, useState } from 'react'
import {
  Container,
  Layout,
  FlexExpander,
  DropDown,
  ButtonVariation,
  Button,
  SelectOption,
  Text
} from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { sortBy } from 'lodash-es'
import { getConfig, getUsingFetch } from 'services/config'
import { useStrings } from 'framework/strings'
import { CodeIcon, GitInfoProps, makeDiffRefs, PullRequestFilterOption } from 'utils/GitUtils'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import type { TypesPrincipalInfo, TypesUser } from 'services/code'
import { useAppContext } from 'AppContext'
import { SearchInputWithSpinner } from 'components/SearchInputWithSpinner/SearchInputWithSpinner'
import { PageBrowserProps, permissionProps } from 'utils/Utils'
import { useQueryParams } from 'hooks/useQueryParams'
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
  const { getString } = useStrings()
  const browserParams = useQueryParams<PageBrowserProps>()
  const [filterOption, setFilterOption] = useState(activePullRequestFilterOption)
  const [authorFilterOption, setAuthorFilterOption] = useState(activePullRequestAuthorFilterOption)
  const [searchTerm, setSearchTerm] = useState('')
  const [query, setQuery] = useState<string>('')
  const [loadingAuthors, setLoadingAuthors] = useState<boolean>(false)
  const space = useGetSpaceParam()
  const { hooks, currentUser, standalone, routingId, routes } = useAppContext()
  const permPushResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY',
        resourceIdentifier: repoMetadata?.uid as string
      },
      permissions: ['code_repo_push']
    },
    [space]
  )

  useEffect(() => {
    setFilterOption(browserParams?.state as string)
  }, [browserParams])

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
  const moveCurrentUserToTop = async (
    authorsList: TypesPrincipalInfo[],
    user: Required<TypesUser>,
    userQuery: string
  ): Promise<TypesPrincipalInfo[]> => {
    const sortedList = sortBy(authorsList, item => item.display_name?.toLowerCase())
    const updateList = (index: number, list: TypesPrincipalInfo[]) => {
      if (index !== -1) {
        const currentUserObj = list[index]
        list.splice(index, 1)
        list.unshift(currentUserObj)
      }
    }
    if (userQuery) return sortedList
    const targetIndex = sortedList.findIndex(obj => obj.uid === user.uid)
    if (targetIndex !== -1) {
      updateList(targetIndex, sortedList)
    } else {
      if (user) {
        const newAuthorsList = await getUsingFetch(getConfig('code/api/v1'), `/principals`, bearerToken, {
          queryParams: {
            query: user?.display_name?.trim(),
            type: 'user',
            accountIdentifier: routingId
          }
        })
        const mergedList = [...new Set(authorsList?.concat(newAuthorsList))]
        const newSortedList = sortBy(mergedList, item => item.display_name?.toLowerCase())
        const newIndex = newSortedList.findIndex(obj => obj.uid === user.uid)
        updateList(newIndex, newSortedList)
        return newSortedList
      }
    }
    return sortedList
  }

  const getAuthorsPromise = async (): Promise<SelectOption[]> => {
    setLoadingAuthors(true)
    try {
      const fetchedAuthors: TypesPrincipalInfo[] = await getUsingFetch(
        getConfig('code/api/v1'),
        `/principals`,
        bearerToken,
        {
          queryParams: {
            query: query?.trim(),
            type: 'user',
            accountIdentifier: routingId
          }
        }
      )
      const authorsList = await moveCurrentUserToTop(fetchedAuthors, currentUser, query)
      const updatedAuthorsList = Array.isArray(authorsList)
        ? ([
            ...(authorsList || []).map(item => ({
              label: JSON.stringify({ displayName: item?.display_name, email: item?.email }),
              value: String(item?.id)
            }))
          ] as SelectOption[])
        : ([] as SelectOption[])
      setLoadingAuthors(false)
      return updatedAuthorsList
    } catch (error) {
      setLoadingAuthors(false)
      throw error
    }
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
          itemRenderer={(item, { handleClick }) => {
            const itemObj = JSON.parse(item.label)
            return (
              <Layout.Horizontal
                padding={{ top: 'small', right: 'small', bottom: 'small', left: 'small' }}
                font={{ variation: FontVariation.BODY }}
                className={css.authorDropdownItem}
                onClick={handleClick}>
                <Text color={Color.GREY_900} className={css.authorName} tooltipProps={{ isDark: true }}>
                  <span>{itemObj.displayName}</span>
                </Text>
                <Text
                  color={Color.GREY_400}
                  font={{ variation: FontVariation.BODY }}
                  lineClamp={1}
                  tooltip={itemObj.email}>
                  ({itemObj.email})
                </Text>
              </Layout.Horizontal>
            )
          }}
          getCustomLabel={item => {
            const itemObj = JSON.parse(item.label)
            return (
              <Layout.Horizontal spacing="small">
                <Text
                  color={Color.GREY_900}
                  font={{ variation: FontVariation.BODY }}
                  tooltip={
                    <Text
                      padding={{ top: 'medium', right: 'medium', bottom: 'medium', left: 'medium' }}
                      color={Color.GREY_0}>
                      {itemObj.email}
                    </Text>
                  }
                  tooltipProps={{ isDark: true }}>
                  {itemObj.displayName}
                </Text>
              </Layout.Horizontal>
            )
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
