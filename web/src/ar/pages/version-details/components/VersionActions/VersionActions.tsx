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

import { RepositoryConfigType } from '@ar/common/types'
import { useAppStore, useBulkDownloadFile, useAllowSoftDelete, useFeatureFlags, useRoutes } from '@ar/hooks'
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
import DownloadVersionMenuItem from './DownloadVersionMenuItem'
import SoftDeleteVersionMenuItem from './SoftDeleteVersionMenuItem'
import ReEvaluateMenuItem from './ReEvaluateMenuItem'

export default function VersionActions({
  data,
  repoKey,
  artifactKey,
  repoType,
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
  const { HAR_DEPENDENCY_FIREWALL } = useFeatureFlags()
  const isBulkDownloadFileEnabled = useBulkDownloadFile()
  const allowSoftDelete = useAllowSoftDelete()
  const isFirewallEnabled = data.firewallMode ? data.firewallMode !== 'ALLOW' : false
  const allowReEvaluate = HAR_DEPENDENCY_FIREWALL && isFirewallEnabled && repoType === RepositoryConfigType.UPSTREAM

  const isAllowed = (action: VersionAction): boolean => {
    if (!allowedActions) return true
    return allowedActions.includes(action)
  }

  if (Array.isArray(allowedActions) && allowedActions.length === 0) return <></>

  return (
    <ActionButton isOpen={open} setOpen={setOpen}>
      {!isCurrentSessionPublic && isAllowed(VersionAction.Delete) && (
        <>
          {allowSoftDelete && (
            <SoftDeleteVersionMenuItem
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
        </>
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
      {!isCurrentSessionPublic && isAllowed(VersionAction.Quarantine) && !data.isQuarantined && digestCount < 2 && (
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
      {!isCurrentSessionPublic && isAllowed(VersionAction.Quarantine) && data.isQuarantined && digestCount < 2 && (
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
      {isBulkDownloadFileEnabled && isAllowed(VersionAction.Download) && (
        <DownloadVersionMenuItem
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
      {allowReEvaluate && isAllowed(VersionAction.ReEvaluate) && (
        <ReEvaluateMenuItem
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
