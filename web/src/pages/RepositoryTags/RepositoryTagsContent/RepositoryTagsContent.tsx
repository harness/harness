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
import { useHistory } from 'react-router-dom'
import type { RepoCommitTag } from 'services/code'
import { usePageIndex } from 'hooks/usePageIndex'
import { LIST_FETCHING_LIMIT, permissionProps, PageBrowserProps } from 'utils/Utils'
import { useQueryParams } from 'hooks/useQueryParams'
import { useUpdateQueryParams } from 'hooks/useUpdateQueryParams'
import { useAppContext } from 'AppContext'
import type { GitInfoProps } from 'utils/GitUtils'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { useShowRequestError } from 'hooks/useShowRequestError'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useStrings } from 'framework/strings'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { useCreateTagModal } from 'components/CreateTagModal/CreateTagModal'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { RepositoryTagsContentHeader } from '../RepositoryTagsContentHeader/RepositoryTagsContentHeader'
import { TagsContent } from '../TagsContent/TagsContent'
import css from './RepositoryTagsContent.module.scss'

export function RepositoryTagsContent({ repoMetadata }: Pick<GitInfoProps, 'repoMetadata'>) {
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const history = useHistory()
  const [searchTerm, setSearchTerm] = useState<string | undefined>()
  const openModal = useCreateTagModal({
    repoMetadata,
    onSuccess: () => {
      refetch()
    },
    showSuccessMessage: true
  })
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
  } = useGet<RepoCommitTag[]>({
    path: `/api/v1/repos/${repoMetadata.path}/+/tags`,
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
  const space = useGetSpaceParam()

  const { standalone } = useAppContext()
  const { hooks } = useAppContext()
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

  return (
    <>
      <LoadingSpinner visible={loading && searchTerm === undefined} />
      <Container padding="xlarge" className={css.resourceContent}>
        <RepositoryTagsContentHeader
          loading={loading && searchTerm !== undefined}
          repoMetadata={repoMetadata}
          onBranchTypeSwitched={gitRef => {
            setPage(1)
            history.push(
              routes.toCODECommits({
                repoPath: repoMetadata.path as string,
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
          <TagsContent
            branches={branches}
            repoMetadata={repoMetadata}
            searchTerm={searchTerm}
            onDeleteSuccess={refetch}
          />
        )}

        <NoResultCard
          permissionProp={permissionProps(permPushResult, standalone)}
          buttonText={getString('newTag')}
          showWhen={() => !!branches && branches.length === 0}
          forSearch={!!searchTerm}
          message={getString('tagEmpty')}
          onButtonClick={() => {
            openModal()
          }}
        />

        <ResourceListingPagination response={response} page={page} setPage={setPage} />
      </Container>
    </>
  )
}
