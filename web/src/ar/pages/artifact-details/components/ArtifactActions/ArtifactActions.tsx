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

import { useAppStore } from '@ar/hooks'
import { PageType } from '@ar/common/types'
import ActionButton from '@ar/components/ActionButton/ActionButton'

import SetupClientMenuItem from './SetupClientMenuItem'
import type { ArtifactActionProps } from './types'
import DeleteArtifactMenuItem from './DeleteArtifactMenuItem'

export default function ArtifactActions({
  data,
  repoKey,
  artifactKey,
  pageType,
  readonly,
  onClose
}: ArtifactActionProps): JSX.Element {
  const [open, setOpen] = useState(false)
  const { isCurrentSessionPublic } = useAppStore()
  return (
    <ActionButton isOpen={open} setOpen={setOpen}>
      {!isCurrentSessionPublic && (
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
      {pageType === PageType.Table && (
        <SetupClientMenuItem
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
