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

import { PageType } from '@ar/common/types'
import ActionButton from '@ar/components/ActionButton/ActionButton'

import type { VersionActionProps } from './types'
import SetupClientMenuItem from './SetupClientMenuItem'
import DeleteVersionMenuItem from './DeleteVersionMenuItem'

export default function VersionActions({
  data,
  repoKey,
  artifactKey,
  versionKey,
  pageType,
  readonly,
  onClose
}: VersionActionProps): JSX.Element {
  const [open, setOpen] = useState(false)
  return (
    <ActionButton isOpen={open} setOpen={setOpen}>
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
      {pageType === PageType.Table && (
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
    </ActionButton>
  )
}
