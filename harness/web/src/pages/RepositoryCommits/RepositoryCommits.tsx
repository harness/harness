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

import React, { useEffect } from 'react'
import { Container, FlexExpander, Layout, PageBody } from '@harnessio/uicore'
import { useHistory } from 'react-router-dom'
import { useGet } from 'restful-react'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useAppContext } from 'AppContext'
import { usePageIndex } from 'hooks/usePageIndex'
import { useQueryParams } from 'hooks/useQueryParams'
import { useUpdateQueryParams } from 'hooks/useUpdateQueryParams'
import type { TypesCommit } from 'services/code'
import { voidFn, getErrorMessage, LIST_FETCHING_LIMIT, PageBrowserProps } from 'utils/Utils'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { BranchTagSelect } from 'components/BranchTagSelect/BranchTagSelect'
import { CommitsView } from 'components/CommitsView/CommitsView'
import { normalizeGitRef } from 'utils/GitUtils'
import css from './RepositoryCommits.module.scss'

export default function RepositoryCommits() {
  const { repoMetadata, error, loading, commitRef, refetch } = useGetRepositoryMetadata()
  const { routes } = useAppContext()
  const history = useHistory()
  const { getString } = useStrings()
  const { updateQueryParams } = useUpdateQueryParams()

  const pageBrowser = useQueryParams<PageBrowserProps>()
  const pageInit = pageBrowser.page ? parseInt(pageBrowser.page) : 1
  const [page, setPage] = usePageIndex(pageInit)
  const {
    data: commits,
    response,
    error: errorCommits,
    loading: loadingCommits
  } = useGet<{ commits: TypesCommit[] }>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/commits`,
    queryParams: {
      limit: LIST_FETCHING_LIMIT,
      page,
      git_ref: normalizeGitRef(commitRef || repoMetadata?.default_branch)
    },
    lazy: !repoMetadata
  })

  useEffect(() => {
    if (pageBrowser.page) {
      updateQueryParams({ page: page.toString() })
    }
  }, [setPage]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('commits')}
        dataTooltipId="repositoryCommits"
      />

      <PageBody error={getErrorMessage(error || errorCommits)} retryOnError={voidFn(refetch)}>
        <LoadingSpinner visible={loading || loadingCommits} withBorder={!!commits && loadingCommits} />

        {(repoMetadata && !!commits?.commits?.length && (
          <Container padding="xlarge" className={css.resourceContent}>
            <Container className={css.contentHeader}>
              <Layout.Horizontal spacing="medium">
                <BranchTagSelect
                  repoMetadata={repoMetadata}
                  disableBranchCreation
                  disableViewAllBranches
                  gitRef={commitRef || (repoMetadata.default_branch as string)}
                  onSelect={ref => {
                    setPage(1)
                    history.push(
                      routes.toCODECommits({
                        repoPath: repoMetadata.path as string,
                        commitRef: `${ref}`
                      })
                    )
                  }}
                />
                <FlexExpander />
              </Layout.Horizontal>
            </Container>

            <CommitsView
              commits={commits?.commits}
              repoMetadata={repoMetadata}
              emptyTitle={getString('noCommits')}
              emptyMessage={getString('noCommitsMessage')}
              showHistoryIcon={true}
            />

            <ResourceListingPagination response={response} page={page} setPage={setPage} />
          </Container>
        )) ||
          null}
      </PageBody>
    </Container>
  )
}
