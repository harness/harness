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

import { useRoutes } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import ActionButton from '@ar/components/ActionButton/ActionButton'
import CopyMenuItem from '@ar/components/MenuItemTypes/CopyMenuItem'
import LinkMenuItem from '@ar/components/MenuItemTypes/LinkMenuItem'

import SetupClientMenuItem from './SetupClientMenuItem'
import DeleteVersionMenuItem from './DeleteVersionMenuItem'
import { type VersionActionProps, VersionAction } from './types'
import { VersionDetailsTab } from '../VersionDetailsTabs/constants'

export default function VersionActions({
  data,
  repoKey,
  artifactKey,
  versionKey,
  pageType,
  readonly,
  onClose,
  allowedActions
}: VersionActionProps): JSX.Element {
  const [open, setOpen] = useState(false)
  const routes = useRoutes()
  const { getString } = useStrings()

  const isAllowed = (action: VersionAction): boolean => {
    if (!allowedActions) return true
    return allowedActions.includes(action)
  }

  if (Array.isArray(allowedActions) && allowedActions.length === 0) return <></>

  return (
    <ActionButton isOpen={open} setOpen={setOpen}>
      {isAllowed(VersionAction.Delete) && (
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
            versionTab: VersionDetailsTab.OVERVIEW
          })}>
          {getString('view')}
        </LinkMenuItem>
      )}
    </ActionButton>
  )
}
