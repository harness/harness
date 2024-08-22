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
import { PageType } from '@ar/common/types'
import RepositoryActionsWrapper from './RepositoryActionsWrapper'
import DeleteRepositoryMenuItem from './DeleteRepository'
import SetupClientMenuItem from './SetupClient'
import type { RepositoryActionsProps } from './types'

export default function RepositoryActions({ data, readonly, pageType }: RepositoryActionsProps): JSX.Element {
  return (
    <RepositoryActionsWrapper data={data} readonly={readonly} pageType={pageType}>
      <DeleteRepositoryMenuItem data={data} readonly={readonly} pageType={pageType} />
      {pageType === PageType.Table && <SetupClientMenuItem data={data} readonly={readonly} pageType={pageType} />}
    </RepositoryActionsWrapper>
  )
}
