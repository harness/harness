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

import React from 'react'
import { Page } from '@harnessio/uicore'
import { useGetArtifactFilesQuery } from '@harnessio/react-har-service-client'

import { DEFAULT_PAGE_INDEX } from '@ar/constants'
import type { VersionDetailsPathParams } from '@ar/routes/types'
import { useDecodedParams, useGetSpaceRef, useParentHooks } from '@ar/hooks'
import {
  type ArtifactFileListPageQueryParams,
  useArtifactFileListQueryParamOptions
} from '@ar/pages/version-details/components/ArtifactFileListTable/utils'
import ArtifactFileListTable from '@ar/pages/version-details/components/ArtifactFileListTable/ArtifactFileListTable'
import versionDetailsPageCss from '../../styles.module.scss'

export default function GenericArtifactDetailsPage() {
  const registryRef = useGetSpaceRef()
  const { useQueryParams, useUpdateQueryParams } = useParentHooks()
  const { updateQueryParams } = useUpdateQueryParams<Partial<ArtifactFileListPageQueryParams>>()

  const pathParams = useDecodedParams<VersionDetailsPathParams>()
  const queryParamOptions = useArtifactFileListQueryParamOptions()
  const queryParams = useQueryParams<ArtifactFileListPageQueryParams>(queryParamOptions)
  const { searchTerm, page, size, sort } = queryParams

  const [sortField, sortOrder] = sort || []

  const {
    isFetching: loading,
    error,
    data,
    refetch
  } = useGetArtifactFilesQuery({
    registry_ref: registryRef,
    artifact: pathParams.artifactIdentifier,
    version: pathParams.versionIdentifier,
    queryParams: {
      searchTerm,
      page,
      size,
      sortField,
      sortOrder
    }
  })
  const response = data?.content

  return (
    <Page.Body
      className={versionDetailsPageCss.pageBody}
      loading={loading}
      error={error?.message || error}
      retryOnError={() => refetch()}>
      {response && (
        <ArtifactFileListTable
          data={response}
          gotoPage={pageNumber => updateQueryParams({ page: pageNumber })}
          setSortBy={sortArr => {
            updateQueryParams({ sort: sortArr, page: DEFAULT_PAGE_INDEX })
          }}
          sortBy={sort}
        />
      )}
    </Page.Body>
  )
}
