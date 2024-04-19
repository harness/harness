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

import React, { useMemo, useRef } from 'react'
import { Container, FlexExpander, Layout, PageBody } from '@harnessio/uicore'
import { useGet } from 'restful-react'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useSetPageContainerWidthVar } from 'hooks/useSetPageContainerWidthVar'
import { useAppContext } from 'AppContext'
import type { TypesCommit } from 'services/code'
import { voidFn, getErrorMessage, LIST_FETCHING_LIMIT } from 'utils/Utils'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { Changes } from 'components/Changes/Changes'
import CommitInfo from 'components/CommitInfo/CommitInfo'
import { normalizeGitRef } from 'utils/GitUtils'
import css from './RepositoryCommit.module.scss'

export default function RepositoryCommits() {
  const { repoMetadata, error, loading, commitRef, refetch } = useGetRepositoryMetadata()
  const { routes, standalone } = useAppContext()
  const { getString } = useStrings()

  const {
    data: commits,
    error: errorCommits,
    loading: loadingCommits
  } = useGet<{ commits: TypesCommit[] }>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/commits`,
    queryParams: {
      limit: LIST_FETCHING_LIMIT,
      git_ref: normalizeGitRef(commitRef || repoMetadata?.default_branch)
    },
    lazy: !repoMetadata
  })

  const ChangesTab = useMemo(() => {
    if (repoMetadata) {
      return (
        <Container className={css.changesContainer}>
          <Changes
            showCommitsDropdown={false}
            readOnly={true}
            repoMetadata={repoMetadata}
            commitSHA={commitRef}
            emptyTitle={getString('noChanges')}
            emptyMessage={getString('noChangesCompare')}
            scrollElement={
              (standalone ? document.querySelector(`.${css.main}`)?.parentElement || window : window) as HTMLElement
            }
          />
        </Container>
      )
    }
  }, [repoMetadata, commitRef, getString, standalone])
  const domRef = useRef<HTMLDivElement>(null)
  useSetPageContainerWidthVar({ domRef })
  return (
    <Container className={css.main} ref={domRef}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('commits')}
        dataTooltipId="repositoryCommits"
        extraBreadcrumbLinks={
          commitRef && repoMetadata
            ? [
                {
                  label: getString('commits'),
                  url: routes.toCODECommits({ repoPath: repoMetadata.path as string, commitRef: '' })
                }
              ]
            : undefined
        }
      />

      <PageBody error={getErrorMessage(error || errorCommits)} retryOnError={voidFn(refetch)}>
        <LoadingSpinner visible={loading || loadingCommits} withBorder={!!commits && loadingCommits} />
        {(repoMetadata && commitRef && !!commits?.commits?.length && (
          <Container padding="xlarge" className={css.resourceContent}>
            <Container className={css.contentHeader}>
              <Layout.Horizontal>
                <CommitInfo repoMetadata={repoMetadata} commitRef={commitRef} />
                <FlexExpander />
              </Layout.Horizontal>
            </Container>
            {ChangesTab}
          </Container>
        )) ||
          null}
      </PageBody>
    </Container>
  )
}
