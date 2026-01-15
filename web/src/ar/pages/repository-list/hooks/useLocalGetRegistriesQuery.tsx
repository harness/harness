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
  type GetAllRegistriesQueryQueryParams,
  useGetAllRegistriesQuery,
  type ListRegistriesOkResponse,
  type ListRegistriesQueryQueryParams,
  type RegistryMetadata,
  useListRegistriesQuery
} from '@harnessio/react-har-service-client'

import { useAppStore, useGetSpaceRef, useParentHooks, useV2Apis } from '@ar/hooks'
import useMetadatadataFilterFromQuery from '@ar/components/MetadataFilterSelector/useMetadataFilterFromQuery'
import { ArtifactRepositoryListPageQueryParams, useArtifactRepositoriesQueryParamOptions } from '../utils'

interface UseLocalGetRegistriesQueryProps extends GetAllRegistriesQueryQueryParams {
  metadata?: string[]
}

export default function useLocalGetRegistriesQuery(
  props: UseLocalGetRegistriesQueryProps
): UseQueryResult<ListRegistriesOkResponse, Error> {
  const spaceRef = useGetSpaceRef()
  const { scope } = useAppStore()
  const shouldUseV2Apis = useV2Apis()
  const { useQueryParams } = useParentHooks()
  const { getValueForAPI } = useMetadatadataFilterFromQuery()

  const queryParamOptions = useArtifactRepositoriesQueryParamOptions()
  const queryParams = useQueryParams<ArtifactRepositoryListPageQueryParams>(queryParamOptions)
  const { softDeleteFilter } = queryParams

  const v1Response = useGetAllRegistriesQuery(
    {
      space_ref: spaceRef,
      queryParams: {
        ...props
      },
      stringifyQueryParamsOptions: {
        arrayFormat: 'repeat'
      }
    },
    {
      enabled: !shouldUseV2Apis
    }
  )

  const v2Response = useListRegistriesQuery(
    {
      queryParams: {
        ...props,
        type: props.type ? props.type : undefined,
        sort_order: props.sort_order as ListRegistriesQueryQueryParams['sort_order'],
        account_identifier: scope.accountId as string,
        org_identifier: scope.orgIdentifier,
        project_identifier: scope.projectIdentifier,
        metadata: getValueForAPI(),
        deleted: softDeleteFilter as ListRegistriesQueryQueryParams['deleted']
      },
      stringifyQueryParamsOptions: {
        arrayFormat: 'repeat'
      }
    },
    {
      enabled: shouldUseV2Apis
    }
  ) as UseQueryResult<ListRegistriesOkResponse, Error>

  const convertedV1ResponseToV2 = useMemo((): UseQueryResult<ListRegistriesOkResponse, Error> => {
    if (!v1Response.data?.content?.data) {
      return v1Response as UseQueryResult<ListRegistriesOkResponse, Error>
    }
    const v1Data = v1Response.data.content.data
    const convertedData: ListRegistriesOkResponse = {
      content: {
        data: {
          registries: (v1Data.registries as RegistryMetadata[]) || [],
          itemCount: v1Data.itemCount,
          pageCount: v1Data.pageCount,
          pageIndex: v1Data.pageIndex,
          pageSize: v1Data.pageSize,
          meta: { activeCount: 0, deletedCount: 0 }
        },
        status: v1Response.data.content.status
      }
    }

    return {
      ...v1Response,
      data: convertedData
    } as UseQueryResult<ListRegistriesOkResponse, Error>
  }, [v1Response])

  return shouldUseV2Apis ? v2Response : convertedV1ResponseToV2
}
