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
import { Text } from '@harnessio/uicore'

import { useDecodedParams } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import type { ManageRegistriesTabPathParams } from '@ar/routes/types'

import { ManageRegistriesDetailsTab } from './constants'

function ManageRegistriesTabs() {
  const { getString } = useStrings()
  const { tab } = useDecodedParams<ManageRegistriesTabPathParams>()
  switch (tab) {
    case ManageRegistriesDetailsTab.LABELS:
      // TODO: Add labels tab
      return <>Labels</>
    case ManageRegistriesDetailsTab.CLEANUP_POLICIES:
      // TODO: Add cleanup policies tab
      return <>Cleanup Policies</>
    default:
      return <Text intent="warning">{getString('stepNotFound')}</Text>
  }
}

export default ManageRegistriesTabs
