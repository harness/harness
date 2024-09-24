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

import React, { useState, createContext } from 'react'
import type { FC, PropsWithChildren } from 'react'
import { defaultTo, noop } from 'lodash-es'
import { Page } from '@harnessio/uicore'
import { useGetRegistryQuery } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'
import { useParentHooks, useDecodedParams, useAppStore, useGetSpaceRef } from '@ar/hooks'
import type { RepositoryDetailsPathParams } from '@ar/routes/types'
import type { Repository } from '@ar/pages/repository-details/types'

export interface RepositoryProviderProps {
  data: Repository | undefined
  isDirty: boolean
  isUpdating: boolean
  setIsDirty: (val: boolean) => void
  setIsLoading: (val: boolean) => void
  setIsUpdating: (val: boolean) => void
  isReadonly: boolean
  refetch: () => void
}

export const RepositoryProviderContext = createContext<RepositoryProviderProps>({
  data: undefined,
  isDirty: false,
  isUpdating: false,
  setIsDirty: noop,
  setIsLoading: noop,
  setIsUpdating: noop,
  isReadonly: false,
  refetch: noop
})

const RepositoryProvider: FC<PropsWithChildren<{ className?: string }>> = ({ children, className }): JSX.Element => {
  const { repositoryIdentifier } = useDecodedParams<RepositoryDetailsPathParams>()
  const [isDirty, setIsDirty] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [isUpdating, setIsUpdating] = useState(false)
  const { usePermission } = useParentHooks()
  const spaceRef = useGetSpaceRef()
  const { scope } = useAppStore()
  const { getString } = useStrings()
  const { accountId, orgIdentifier, projectIdentifier } = scope

  const [isEdit] = usePermission(
    {
      resourceScope: {
        accountIdentifier: accountId,
        orgIdentifier,
        projectIdentifier
      },
      resource: {
        resourceType: ResourceType.ARTIFACT_REGISTRY,
        resourceIdentifier: repositoryIdentifier
      },
      permissions: [PermissionIdentifier.EDIT_ARTIFACT_REGISTRY]
    },
    [accountId, projectIdentifier, orgIdentifier, repositoryIdentifier]
  )

  const {
    data: repositoryData,
    isError,
    error,
    isFetching: isDataLoading,
    refetch
  } = useGetRegistryQuery({
    registry_ref: spaceRef
  })
  const loading = isLoading || isDataLoading
  return (
    <RepositoryProviderContext.Provider
      value={{
        data: repositoryData?.content?.data as RepositoryProviderProps['data'],
        isDirty,
        isUpdating,
        setIsDirty,
        setIsLoading,
        setIsUpdating,
        isReadonly: !isEdit,
        refetch
      }}>
      <Page.Body
        className={className}
        loading={loading}
        error={isError && defaultTo(error?.message, getString('failedToLoadData'))}
        retryOnError={() => refetch()}>
        {loading ? null : children}
      </Page.Body>
    </RepositoryProviderContext.Provider>
  )
}

export default RepositoryProvider
