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
import { useCreateBulkDownloadRequestMutation } from '@harnessio/react-har-service-v2-client'

import { encodeFileName } from '@ar/common/utils'
import { useAppStore, useParentComponents } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'
import { AsyncDownloadRequestsContext } from '@ar/contexts/AsyncDownloadRequestsProvider/AsyncDownloadRequestsProvider'

import type { ArtifactActionProps } from './types'

export default function DownloadArtifactMenuItem(props: ArtifactActionProps) {
  const { artifactKey, readonly, onClose, data } = props
  const { RbacMenuItem } = useParentComponents()
  const { scope } = useAppStore()
  const { getString } = useStrings()
  const { addRequest } = useContext(AsyncDownloadRequestsContext)

  const { showSuccess, showError } = useToaster()

  const { mutateAsync: createBulkDownloadRequest } = useCreateBulkDownloadRequestMutation()

  const handleDownload = async () => {
    try {
      const response = await createBulkDownloadRequest({
        body: {
          packages: [data.uuid],
          registryId: data.registryUUID,
          outputFileName: encodeFileName(`${artifactKey}.zip`)
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

  return (
    <RbacMenuItem
      icon="download-box"
      text={getString('artifactList.table.actions.download')}
      onClick={handleDownload}
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
