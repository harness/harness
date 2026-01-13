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

import { useMemo } from 'react'
import { useParams } from 'react-router-dom'
import type { UseQueryResult } from '@tanstack/react-query'
import {
  useGetAllArtifactVersionsQuery,
  GetAllArtifactVersionsQueryQueryParams,
  type ListVersionsOkResponse,
  type ListVersionsQueryQueryParams,
  useListVersionsQuery,
  VersionMetadata
} from '@harnessio/react-har-service-client'

import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import { useGetSpaceRef, useAppStore, useV2Apis, useParentHooks } from '@ar/hooks'
import type { ArtifactDetailsPathParams } from '@ar/routes/types'
import useMetadatadataFilterFromQuery from '@ar/components/MetadataFilterSelector/useMetadataFilterFromQuery'
import { useVersionListQueryParamOptions, VersionListPageQueryParams } from '../utils'

const COLUMN_NAME_MAPPING_FROM_V2_TO_V1: Record<string, string> = {
  version: 'name'
}

interface UseLocalGetAllArtifactVersionsQueryProps extends GetAllArtifactVersionsQueryQueryParams {
  metadata?: string[]
}

function useLocalGetAllArtifactVersionsQuery(props: UseLocalGetAllArtifactVersionsQueryProps) {
  const registryRef = useGetSpaceRef()
  const shouldUseV2Apis = useV2Apis()
  const { scope } = useAppStore()
  const pathParams = useParams<ArtifactDetailsPathParams>()
  const { getValueForAPI } = useMetadatadataFilterFromQuery()

  const { useQueryParams } = useParentHooks()
  const queryParams = useQueryParams<VersionListPageQueryParams>(useVersionListQueryParamOptions())
  const { softDeleteFilter } = queryParams

  const v1Response = useGetAllArtifactVersionsQuery(
    {
      registry_ref: registryRef,
      artifact: encodeRef(pathParams.artifactIdentifier),
      queryParams: {
        ...props,
        sort_field: props.sort_field
          ? COLUMN_NAME_MAPPING_FROM_V2_TO_V1[props.sort_field] ?? props.sort_field
          : undefined
      },
      stringifyQueryParamsOptions: {
        arrayFormat: 'repeat'
      }
    },
    {
      enabled: !shouldUseV2Apis
    }
  )

  const v2Response = useListVersionsQuery(
    {
      queryParams: {
        ...props,
        account_identifier: scope.accountId as string,
        org_identifier: scope.orgIdentifier,
        project_identifier: scope.projectIdentifier,
        registry_identifier: [pathParams.repositoryIdentifier],
        package: pathParams.artifactIdentifier,
        sort_order: props.sort_order as ListVersionsQueryQueryParams['sort_order'],
        metadata: getValueForAPI(),
        deleted: softDeleteFilter as ListVersionsQueryQueryParams['deleted']
      },
      stringifyQueryParamsOptions: {
        arrayFormat: 'repeat'
      }
    },
    {
      enabled: shouldUseV2Apis
    }
  )

  const convertedV1ResponseToV2: UseQueryResult<ListVersionsOkResponse, Error> = useMemo(() => {
    if (!v1Response.data?.content?.data) {
      return v1Response as UseQueryResult<ListVersionsOkResponse, Error>
    }
    const v1Data = v1Response.data.content.data
    const convertedData: ListVersionsOkResponse = {
      content: {
        data: {
          artifacts:
            v1Data.artifactVersions?.map(
              artifact =>
                ({
                  ...artifact,
                  version: artifact.name,
                  package: pathParams.artifactIdentifier,
                  lastModified: artifact.lastModified as string,
                  isDeleted: false
                } as VersionMetadata)
            ) || [],
          itemCount: v1Data.itemCount,
          pageCount: v1Data.pageCount,
          pageIndex: v1Data.pageIndex,
          pageSize: v1Data.pageSize,
          meta: {
            activeCount: 0,
            deletedCount: 0
          }
        },
        status: v1Response.data.content.status
      }
    }

    return {
      ...v1Response,
      data: convertedData
    } as UseQueryResult<ListVersionsOkResponse, Error>
  }, [v1Response])

  return shouldUseV2Apis ? v2Response : convertedV1ResponseToV2
}

export default useLocalGetAllArtifactVersionsQuery
