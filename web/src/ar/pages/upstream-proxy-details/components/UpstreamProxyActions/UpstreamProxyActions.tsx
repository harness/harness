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
import ActionButton from '@ar/components/ActionButton/ActionButton'

import { PageType } from '@ar/common/types'
import { useAllowSoftDelete } from '@ar/hooks'
import SetupClientMenuItem from '@ar/pages/repository-details/components/Actions/SetupClient'
import DeleteRepositoryMenuItem from '@ar/pages/repository-details/components/Actions/DeleteRepository'
import SoftDeleteRepositoryMenuItem from '@ar/pages/repository-details/components/Actions/SoftDeleteRepository'

import type { UpstreamProxyActionProps } from './type'

export default function UpstreamProxyActions({ data, readonly, pageType }: UpstreamProxyActionProps): JSX.Element {
  const [open, setOpen] = useState(false)
  const allowSoftDelete = useAllowSoftDelete()
  return (
    <ActionButton isOpen={open} setOpen={setOpen}>
      {allowSoftDelete && (
        <SoftDeleteRepositoryMenuItem
          data={data}
          readonly={readonly}
          pageType={pageType}
          onClose={() => setOpen(false)}
        />
      )}
      <DeleteRepositoryMenuItem data={data} readonly={readonly} pageType={pageType} onClose={() => setOpen(false)} />
      {pageType === PageType.Table && (
        <SetupClientMenuItem data={data} readonly={readonly} pageType={pageType} onClose={() => setOpen(false)} />
      )}
    </ActionButton>
  )
}
