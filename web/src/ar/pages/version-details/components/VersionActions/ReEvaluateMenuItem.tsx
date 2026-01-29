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
import { useToaster } from '@harnessio/uicore'
import { evaluateArtifactScan, type V3Error } from '@harnessio/react-har-service-client'

import { useAppStore, useParentComponents } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { queryClient } from '@ar/utils/queryClient'
import { ResourceType } from '@ar/common/permissionTypes'
import { PermissionIdentifier } from '@ar/common/permissionTypes'

import type { VersionActionProps } from './types'

function ReEvaluateMenuItem(props: VersionActionProps) {
  const { repoKey, readonly, onClose, data } = props
  const { getString } = useStrings()
  const { RbacMenuItem } = useParentComponents()
  const { scope } = useAppStore()
  const { accountId } = scope
  const { clear, showSuccess, showError } = useToaster()

  const handleAfterReScan = () => {
    queryClient.invalidateQueries(['GetAllArtifactVersions'])
    queryClient.invalidateQueries(['GetAllHarnessArtifacts'])
    queryClient.invalidateQueries(['GetArtifactVersionSummary'])
    onClose?.()
  }

  const handleReEvaluate = async () => {
    return evaluateArtifactScan({
      queryParams: { account_identifier: accountId || '' },
      body: { versionId: data.uuid }
    })
      .then(() => {
        clear()
        showSuccess(getString('versionList.messages.reEvaluateSuccess'))
      })
      .catch((err: V3Error) => {
        clear()
        showError(err?.error?.message ?? getString('versionList.messages.reEvaluateFailed'))
      })
      .finally(() => {
        handleAfterReScan()
      })
  }

  return (
    <RbacMenuItem
      icon="refresh"
      text={getString('versionList.actions.reEvaluate')}
      onClick={handleReEvaluate}
      disabled={readonly}
      permission={{
        resource: {
          resourceType: ResourceType.ARTIFACT_REGISTRY,
          resourceIdentifier: repoKey
        },
        permission: PermissionIdentifier.UPLOAD_ARTIFACT
      }}
    />
  )
}

export default ReEvaluateMenuItem
