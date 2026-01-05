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

import React, { useState } from 'react'

import { useAppStore, useBulkDownloadFile } from '@ar/hooks'
import { PageType } from '@ar/common/types'
import ActionButton from '@ar/components/ActionButton/ActionButton'

import SetupClientMenuItem from './SetupClientMenuItem'
import { ArtifactActionProps, ArtifactActionsEnum } from './types'
import DeleteArtifactMenuItem from './DeleteArtifactMenuItem'
import DownloadArtifactMenuItem from './DownloadArtifactMenuItem'

export default function ArtifactActions({
  data,
  repoKey,
  artifactKey,
  pageType,
  readonly,
  onClose,
  allowedActions
}: ArtifactActionProps): JSX.Element {
  const [open, setOpen] = useState(false)
  const { isCurrentSessionPublic } = useAppStore()
  const isBulkDownloadFileEnabled = useBulkDownloadFile()

  const isSupportedAction = (action: ArtifactActionsEnum) => {
    if (!allowedActions) {
      return true
    }
    return allowedActions.includes(action)
  }
  return (
    <ActionButton isOpen={open} setOpen={setOpen}>
      {!isCurrentSessionPublic && isSupportedAction(ArtifactActionsEnum.Delete) && (
        <DeleteArtifactMenuItem
          artifactKey={artifactKey}
          repoKey={repoKey}
          data={data}
          pageType={pageType}
          readonly={readonly}
          onClose={() => {
            setOpen(false)
            onClose?.()
          }}
        />
      )}
      {pageType === PageType.Table && isSupportedAction(ArtifactActionsEnum.SetupClient) && (
        <SetupClientMenuItem
          data={data}
          pageType={pageType}
          readonly={readonly}
          onClose={() => setOpen(false)}
          artifactKey={artifactKey}
          repoKey={repoKey}
        />
      )}
      {isBulkDownloadFileEnabled && isSupportedAction(ArtifactActionsEnum.Download) && (
        <DownloadArtifactMenuItem
          data={data}
          pageType={pageType}
          readonly={readonly}
          onClose={() => setOpen(false)}
          artifactKey={artifactKey}
          repoKey={repoKey}
        />
      )}
    </ActionButton>
  )
}
