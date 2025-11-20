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

import { type GetAllRegistriesQueryQueryParams, useGetAllRegistriesQuery } from '@harnessio/react-har-service-client'
import { type ListRegistriesQueryQueryParams, useListRegistriesQuery } from '@harnessio/react-har-service-v2-client'

import { useAppStore, useGetSpaceRef, useV2Apis } from '@ar/hooks'
import useMetadatadataFilterFromQuery from '@ar/components/MetadataFilterSelector/useMetadataFilterFromQuery'

interface UseLocalGetRegistriesQueryProps extends GetAllRegistriesQueryQueryParams {
  metadata?: string[]
}

export default function useLocalGetRegistriesQuery(props: UseLocalGetRegistriesQueryProps) {
  const spaceRef = useGetSpaceRef()
  const { scope } = useAppStore()
  const shouldUseV2Apis = useV2Apis()
  const { getValueForAPI } = useMetadatadataFilterFromQuery()

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

  return shouldUseV2Apis ? v2Response : v1Response
}
