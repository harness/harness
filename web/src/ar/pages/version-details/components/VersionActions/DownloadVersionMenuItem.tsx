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

import React, { useContext } from 'react'
import { getErrorInfoFromErrorObject, useToaster } from '@harnessio/uicore'
import { createBulkDownloadRequestV1 } from '@harnessio/react-har-service-client'
import { useCreateBulkDownloadRequestMutation } from '@harnessio/react-har-service-client'

import { encodeFileName } from '@ar/common/utils'
import { useAppStore, useGetSpaceRef, useParentComponents, useV2Apis } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import { RepositoryPackageType } from '@ar/common/types'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'
import { AsyncDownloadRequestsContext } from '@ar/contexts/AsyncDownloadRequestsProvider/AsyncDownloadRequestsProvider'

import type { VersionActionProps } from './types'

export default function DownloadVersionMenuItem(props: VersionActionProps) {
  const { artifactKey, repoKey, versionKey, readonly, onClose, data } = props
  const { RbacMenuItem } = useParentComponents()
  const { scope } = useAppStore()
  const { getString } = useStrings()
  const { addRequest } = useContext(AsyncDownloadRequestsContext)
  const shouldUseV2Apis = useV2Apis()
  const registryRef = useGetSpaceRef(repoKey)
  const isOCIPackageType = [RepositoryPackageType.DOCKER, RepositoryPackageType.HELM].includes(
    data.packageType as RepositoryPackageType
  )

  const { showSuccess, showError } = useToaster()

  const { mutateAsync: createBulkDownloadRequest } = useCreateBulkDownloadRequestMutation()

  const handleDownload = async () => {
    try {
      const response = await createBulkDownloadRequest({
        body: {
          versions: [data.uuid],
          registryId: data.registryUUID,
          outputFileName: encodeFileName(`${artifactKey}/${versionKey}.zip`)
        },
        queryParams: {
          account_identifier: scope.accountId as string
        }
      })
      const downloadId = response.content.data.downloadId
      showSuccess(getString('artifactDetails.createdAsyncDownloadRequest'))
      addRequest(downloadId, data.registryUUID)
      onClose?.()
    } catch (error) {
      showError(getErrorInfoFromErrorObject(error as Error) || getString('failedToLoadData'))
    }
  }

  const handleDownloadV1 = async () => {
    try {
      const response = await createBulkDownloadRequestV1({
        registry_ref: registryRef,
        artifact: encodeRef(artifactKey),
        version: versionKey
      })
      const downloadId = response.content.data.downloadId
      showSuccess(getString('artifactDetails.createdAsyncDownloadRequest'))
      addRequest(downloadId, data.registryUUID)
      onClose?.()
    } catch (error) {
      showError(getErrorInfoFromErrorObject(error as Error) || getString('failedToLoadData'))
    }
  }

  return (
    <RbacMenuItem
      icon="download-box"
      text={getString('artifactList.table.actions.download')}
      onClick={() => {
        if (shouldUseV2Apis || !isOCIPackageType) {
          handleDownload()
        } else {
          handleDownloadV1()
        }
      }}
      disabled={readonly}
      permission={{
        resource: {
          resourceType: ResourceType.ARTIFACT_REGISTRY,
          resourceIdentifier: artifactKey
        },
        permission: PermissionIdentifier.DOWNLOAD_ARTIFACT
      }}
    />
  )
}
