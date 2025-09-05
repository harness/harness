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

import React, { useMemo } from 'react'

import { Parent } from '@ar/common/types'
import { useAppStore, useFeatureFlags } from '@ar/hooks'

import DockerTagListTable from './DockerTagListTable'
import type { DockerVersionListTableProps } from './types'
import DockerDigestListTable from './DockerDigestListTable'

function DockerVersionListTable(props: DockerVersionListTableProps): JSX.Element {
  const { HAR_ENABLE_UNTAGGED_IMAGES_SUPPORT } = useFeatureFlags()
  const { parent } = useAppStore()
  const TableComponent = useMemo(() => {
    if (HAR_ENABLE_UNTAGGED_IMAGES_SUPPORT || parent === Parent.OSS) {
      return DockerDigestListTable
    }
    return DockerTagListTable
  }, [HAR_ENABLE_UNTAGGED_IMAGES_SUPPORT, parent])

  return <TableComponent {...props} />
}

export default DockerVersionListTable
