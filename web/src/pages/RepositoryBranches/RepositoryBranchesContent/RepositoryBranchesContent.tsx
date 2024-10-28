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

import React, { useEffect, useState } from 'react'
import { Container } from '@harnessio/uicore'
import { useGet } from 'restful-react'
import { Render } from 'react-jsx-match'
import { useHistory } from 'react-router-dom'
import type { TypesBranchExtended, RepoRepositoryOutput } from 'services/code'
import { usePageIndex } from 'hooks/usePageIndex'
import { LIST_FETCHING_LIMIT, PageBrowserProps } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import type { GitInfoProps } from 'utils/GitUtils'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { useShowRequestError } from 'hooks/useShowRequestError'
import { useQueryParams } from 'hooks/useQueryParams'
import { useUpdateQueryParams } from 'hooks/useUpdateQueryParams'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { BranchesContentHeader } from './BranchesContentHeader/BranchesContentHeader'
import { BranchesContent } from './BranchesContent/BranchesContent'
import css from './RepositoryBranchesContent.module.scss'

export function RepositoryBranchesContent({ repoMetadata }: Partial<Pick<GitInfoProps, 'repoMetadata'>>) {
  const { routes } = useAppContext()
  const history = useHistory()
  const [searchTerm, setSearchTerm] = useState<string | undefined>()
  const { updateQueryParams } = useUpdateQueryParams()
  const pageBrowser = useQueryParams<PageBrowserProps>()
  const pageInit = pageBrowser.page ? parseInt(pageBrowser.page) : 1
  const [page, setPage] = usePageIndex(pageInit)
  const {
    data: branches,
    response,
    error,
    loading,
    refetch
  } = useGet<TypesBranchExtended[]>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/branches`,
    queryParams: {
      limit: LIST_FETCHING_LIMIT,
      page,
      sort: 'date',
      order: 'desc',
      include_commit: true,
      query: searchTerm
    },
    debounce: 500
  })

  useEffect(() => {
    if (page > 1) {
      updateQueryParams({ page: page.toString() })
    }
  }, [setPage]) // eslint-disable-line react-hooks/exhaustive-deps

  useShowRequestError(error)

  return (
    <>
      <LoadingSpinner visible={!repoMetadata || (loading && searchTerm === undefined)} />
      <Render when={repoMetadata}>
        <Container padding="xlarge" className={css.resourceContent}>
          <BranchesContentHeader
            loading={loading && searchTerm !== undefined}
            repoMetadata={repoMetadata as RepoRepositoryOutput}
            onBranchTypeSwitched={gitRef => {
              setPage(1)
              history.push(
                routes.toCODECommits({
                  repoPath: repoMetadata?.path as string,
                  commitRef: gitRef
                })
              )
            }}
            onSearchTermChanged={value => {
              setSearchTerm(value)
              setPage(1)
            }}
            onNewBranchCreated={refetch}
          />

          {!!branches?.length && (
            <BranchesContent
              branches={branches}
              repoMetadata={repoMetadata as RepoRepositoryOutput}
              searchTerm={searchTerm}
              onDeleteSuccess={refetch}
            />
          )}

          <NoResultCard showWhen={() => !!branches && branches.length === 0 && !!searchTerm?.length} forSearch={true} />

          <ResourceListingPagination response={response} page={page} setPage={setPage} />
        </Container>
      </Render>
    </>
  )
}
