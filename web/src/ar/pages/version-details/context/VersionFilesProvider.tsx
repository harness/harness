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

import React, { createContext, useState, type PropsWithChildren } from 'react'
import { PageError, PageSpinner } from '@harnessio/uicore'
import { type FileDetailResponseResponse, useGetArtifactFilesQuery } from '@harnessio/react-har-service-client'

import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import type { VersionDetailsPathParams } from '@ar/routes/types'
import { LocalArtifactType } from '@ar/pages/repository-details/constants'
import { useDecodedParams, useGetSpaceRef, useParentHooks } from '@ar/hooks'
import type { UseUpdateQueryParamsReturn } from '@ar/__mocks__/hooks/useUpdateQueryParams'
import { DEFAULT_ARTIFACT_LIST_TABLE_SORT, DEFAULT_PAGE_INDEX, DEFAULT_PAGE_SIZE } from '@ar/constants'

import {
  ArtifactFileListPageQueryParams,
  useArtifactFileListQueryParamOptions
} from '../components/ArtifactFileListTable/utils'

interface VersionFilesProviderProps {
  data: FileDetailResponseResponse
  updateQueryParams: UseUpdateQueryParamsReturn<Partial<ArtifactFileListPageQueryParams>>['updateQueryParams']
  queryParams: Partial<ArtifactFileListPageQueryParams>
  refetch: () => void
  sort: string[]
}

export const VersionFilesContext = createContext<VersionFilesProviderProps>({} as VersionFilesProviderProps)

interface IVersionFilesProviderProps {
  shouldUseLocalParams?: boolean
  repositoryIdentifier?: string
  artifactIdentifier?: string
  versionIdentifier?: string
  artifactType?: LocalArtifactType
}

const VersionFilesProvider = (props: PropsWithChildren<IVersionFilesProviderProps>) => {
  const { shouldUseLocalParams, artifactIdentifier, versionIdentifier, repositoryIdentifier, artifactType } = props
  const [localParams, setLocalParams] = useState<Partial<ArtifactFileListPageQueryParams>>({
    page: DEFAULT_PAGE_INDEX,
    size: DEFAULT_PAGE_SIZE,
    sort: DEFAULT_ARTIFACT_LIST_TABLE_SORT
  })
  const pathParams = useDecodedParams<VersionDetailsPathParams>()
  const spaceRef = useGetSpaceRef(repositoryIdentifier)

  const { useQueryParams, useUpdateQueryParams } = useParentHooks()
  const { updateQueryParams } = useUpdateQueryParams<Partial<ArtifactFileListPageQueryParams>>()

  const queryParamOptions = useArtifactFileListQueryParamOptions()
  const queryParams = useQueryParams<ArtifactFileListPageQueryParams>(queryParamOptions)
  const { page, size, sort } = shouldUseLocalParams ? localParams : queryParams

  const [sortField, sortOrder] = sort || []

  const transformedArtifactType = artifactType ?? pathParams.artifactType

  const {
    data,
    isFetching: loading,
    error,
    refetch
  } = useGetArtifactFilesQuery({
    registry_ref: spaceRef,
    artifact: encodeRef(artifactIdentifier ?? pathParams.artifactIdentifier),
    version: versionIdentifier ?? pathParams.versionIdentifier,
    queryParams: {
      page,
      size,
      sort_field: sortField,
      sort_order: sortOrder,
      artifact_type: transformedArtifactType === LocalArtifactType.ARTIFACTS ? undefined : transformedArtifactType
    }
  })
  const responseData = data?.content

  return (
    <>
      {loading ? <PageSpinner /> : null}
      {error && !loading ? <PageError message={error.message} onClick={() => refetch()} /> : null}
      {!error && !loading && responseData ? (
        <VersionFilesContext.Provider
          value={{
            data: responseData,
            refetch,
            updateQueryParams: shouldUseLocalParams ? setLocalParams : updateQueryParams,
            queryParams: shouldUseLocalParams ? localParams : queryParams,
            sort: sort || []
          }}>
          {props.children}
        </VersionFilesContext.Provider>
      ) : null}
    </>
  )
}

export default VersionFilesProvider
