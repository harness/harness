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
import { get } from 'lodash-es'

import { useAppStore, useFeatureFlags, useRoutes } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import ActionButton from '@ar/components/ActionButton/ActionButton'
import CopyMenuItem from '@ar/components/MenuItemTypes/CopyMenuItem'
import LinkMenuItem from '@ar/components/MenuItemTypes/LinkMenuItem'
import { LocalArtifactType } from '@ar/pages/repository-details/constants'

import QuarantineMenuItem from './QuarantineMenuItem'
import SetupClientMenuItem from './SetupClientMenuItem'
import DeleteVersionMenuItem from './DeleteVersionMenuItem'
import { type VersionActionProps, VersionAction } from './types'
import { VersionDetailsTab } from '../VersionDetailsTabs/constants'
import RemoveQurantineMenuItem from './RemoveQurantineMenuItem'

export default function VersionActions({
  data,
  repoKey,
  artifactKey,
  versionKey,
  pageType,
  readonly,
  onClose,
  digest,
  digestCount = 0,
  allowedActions
}: VersionActionProps): JSX.Element {
  const [open, setOpen] = useState(false)
  const routes = useRoutes()
  const { isCurrentSessionPublic } = useAppStore()
  const { getString } = useStrings()
  const { HAR_ARTIFACT_QUARANTINE_ENABLED } = useFeatureFlags()

  const isAllowed = (action: VersionAction): boolean => {
    if (!allowedActions) return true
    return allowedActions.includes(action)
  }

  if (Array.isArray(allowedActions) && allowedActions.length === 0) return <></>

  return (
    <ActionButton isOpen={open} setOpen={setOpen}>
      {!isCurrentSessionPublic && isAllowed(VersionAction.Delete) && (
        <DeleteVersionMenuItem
          artifactKey={artifactKey}
          repoKey={repoKey}
          versionKey={versionKey}
          data={data}
          pageType={pageType}
          readonly={readonly}
          onClose={() => {
            setOpen(false)
            onClose?.()
          }}
        />
      )}
      {isAllowed(VersionAction.SetupClient) && (
        <SetupClientMenuItem
          data={data}
          pageType={pageType}
          readonly={readonly}
          onClose={() => setOpen(false)}
          versionKey={versionKey}
          artifactKey={artifactKey}
          repoKey={repoKey}
        />
      )}
      {isAllowed(VersionAction.DownloadCommand) && (
        <CopyMenuItem value={get(data, 'pullCommand', '')} onCopy={() => setOpen(false)} />
      )}
      {isAllowed(VersionAction.ViewVersionDetails) && (
        <LinkMenuItem
          to={routes.toARVersionDetailsTab({
            repositoryIdentifier: repoKey,
            artifactIdentifier: artifactKey,
            versionIdentifier: versionKey,
            versionTab: VersionDetailsTab.OVERVIEW,
            artifactType: (data?.artifactType ?? LocalArtifactType.ARTIFACTS) as LocalArtifactType
          })}>
          {getString('view')}
        </LinkMenuItem>
      )}
      {!isCurrentSessionPublic &&
        isAllowed(VersionAction.Quarantine) &&
        HAR_ARTIFACT_QUARANTINE_ENABLED &&
        !data.isQuarantined &&
        digestCount < 2 && (
          <QuarantineMenuItem
            artifactKey={artifactKey}
            repoKey={repoKey}
            versionKey={digest ?? versionKey}
            data={data}
            pageType={pageType}
            readonly={readonly}
            onClose={() => {
              setOpen(false)
              onClose?.()
            }}
          />
        )}
      {!isCurrentSessionPublic &&
        isAllowed(VersionAction.Quarantine) &&
        HAR_ARTIFACT_QUARANTINE_ENABLED &&
        data.isQuarantined &&
        digestCount < 2 && (
          <RemoveQurantineMenuItem
            artifactKey={artifactKey}
            repoKey={repoKey}
            versionKey={digest ?? versionKey}
            data={data}
            pageType={pageType}
            readonly={readonly}
            onClose={() => {
              setOpen(false)
              onClose?.()
            }}
          />
        )}
    </ActionButton>
  )
}
