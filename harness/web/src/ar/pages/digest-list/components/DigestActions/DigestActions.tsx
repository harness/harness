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

import { useFeatureFlags } from '@ar/hooks'
import ActionButton from '@ar/components/ActionButton/ActionButton'

import QuarantineMenuItem from './QuarantineMenuItem'
import type { DigestActionProps } from './types'
import RemoveQurantineMenuItem from './RemoveQurantineMenuItem'

export default function DigestActions({
  data,
  repoKey,
  artifactKey,
  versionKey,
  readonly,
  pageType,
  onClose
}: DigestActionProps): JSX.Element {
  const [open, setOpen] = useState(false)
  const { HAR_ARTIFACT_QUARANTINE_ENABLED } = useFeatureFlags()

  if (!HAR_ARTIFACT_QUARANTINE_ENABLED) return <></>

  return (
    <ActionButton isOpen={open} setOpen={setOpen}>
      {!data.isQuarantined && (
        <QuarantineMenuItem
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
      {data.isQuarantined && (
        <RemoveQurantineMenuItem
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
    </ActionButton>
  )
}
