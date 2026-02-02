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

import React, { useState, createContext, useEffect } from 'react'
import type { FC, PropsWithChildren } from 'react'
import { noop } from 'lodash-es'
import { PageError, PageSpinner } from '@harnessio/uicore'
import { useGetRegistryQuery } from '@harnessio/react-har-service-client'

import { Parent } from '@ar/common/types'
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

const RepositoryProvider: FC<PropsWithChildren<unknown>> = ({ children }): JSX.Element => {
  const { repositoryIdentifier } = useDecodedParams<RepositoryDetailsPathParams>()
  const [isDirty, setIsDirty] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [isUpdating, setIsUpdating] = useState(false)
  const { usePermission } = useParentHooks()
  const spaceRef = useGetSpaceRef()
  const { scope, updateAppStore, parent } = useAppStore()
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
    error,
    isFetching: isDataLoading,
    refetch
  } = useGetRegistryQuery({
    registry_ref: spaceRef
  })

  const loading = isLoading || isDataLoading

  useEffect(() => {
    if (typeof updateAppStore === 'function' && parent === Parent.OSS && repositoryData) {
      updateAppStore({
        repositoryType: repositoryData.content.data.config?.type,
        repositoryPackageType: repositoryData.content.data.packageType
      })
    }
  }, [repositoryData, loading])

  const data = repositoryData?.content?.data
  const isDeleted = !!data?.deletedAt

  return (
    <RepositoryProviderContext.Provider
      value={{
        data: data as Repository,
        isDirty,
        isUpdating,
        setIsDirty,
        setIsLoading,
        setIsUpdating,
        isReadonly: !isEdit || isDeleted,
        refetch
      }}>
      {loading ? <PageSpinner /> : null}
      {error && !loading ? (
        <PageError message={error.message || getString('failedToLoadData')} onClick={() => refetch()} />
      ) : null}
      {!error && !loading ? children : null}
    </RepositoryProviderContext.Provider>
  )
}

export default RepositoryProvider
