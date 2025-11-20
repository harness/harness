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
import type { UseQueryResult } from '@tanstack/react-query'
import {
  type GetAllHarnessArtifactsQueryQueryParams,
  useGetAllHarnessArtifactsQuery
} from '@harnessio/react-har-service-client'
import {
  type ArtifactMetadata,
  type ListArtifactsOkResponse,
  type ListArtifactsQueryQueryParams,
  useListArtifactsQuery
} from '@harnessio/react-har-service-v2-client'

import { useAppStore, useGetSpaceRef, useV2Apis } from '@ar/hooks'
import useMetadatadataFilterFromQuery from '@ar/components/MetadataFilterSelector/useMetadataFilterFromQuery'

export const COLUMN_NAME_MAPPING_FROM_V2_TO_V1: Record<string, string> = {
  package: 'name'
}

interface UseLocalGetAllHarnessArtifactsQueryProps extends GetAllHarnessArtifactsQueryQueryParams {
  metadata?: string[]
}

export default function useLocalGetAllHarnessArtifactsQuery(props: UseLocalGetAllHarnessArtifactsQueryProps) {
  const spaceRef = useGetSpaceRef()
  const { scope } = useAppStore()
  const shouldUseV2Apis = useV2Apis()
  const { getValueForAPI } = useMetadatadataFilterFromQuery()

  const v1Response = useGetAllHarnessArtifactsQuery(
    {
      space_ref: spaceRef,
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

  const v2Response = useListArtifactsQuery(
    {
      queryParams: {
        ...props,
        account_identifier: scope.accountId as string,
        org_identifier: scope.orgIdentifier,
        project_identifier: scope.projectIdentifier,
        registry_identifier: props.reg_identifier,
        sort_order: props.sort_order as ListArtifactsQueryQueryParams['sort_order'],
        metadata: getValueForAPI()
      },
      stringifyQueryParamsOptions: {
        arrayFormat: 'repeat'
      }
    },
    {
      enabled: shouldUseV2Apis
    }
  )

  const convertedV1ResponseToV2: UseQueryResult<ListArtifactsOkResponse, Error> = useMemo(() => {
    if (!v1Response.data?.content?.data) {
      return v1Response as UseQueryResult<ListArtifactsOkResponse, Error>
    }
    const v1Data = v1Response.data.content.data
    const convertedData: ListArtifactsOkResponse = {
      content: {
        data: {
          artifacts:
            v1Data.artifacts?.map(artifact => {
              const { name, version, ...rest } = artifact
              return {
                ...rest,
                version: version,
                package: name,
                lastModified: artifact.lastModified as string
              } as ArtifactMetadata
            }) || [],
          itemCount: v1Data.itemCount,
          pageCount: v1Data.pageCount,
          pageIndex: v1Data.pageIndex,
          pageSize: v1Data.pageSize
        },
        status: v1Response.data.content.status
      }
    }

    return {
      ...v1Response,
      data: convertedData
    } as UseQueryResult<ListArtifactsOkResponse, Error>
  }, [v1Response])

  return shouldUseV2Apis ? v2Response : convertedV1ResponseToV2
}
