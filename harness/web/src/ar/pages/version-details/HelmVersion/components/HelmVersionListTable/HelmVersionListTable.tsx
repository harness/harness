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
import type { VersionListTableProps } from '@ar/frameworks/Version/Version'
import { useAppStore, useFeatureFlags } from '@ar/hooks'
import { Parent } from '@ar/common/types'
import VersionListTable, {
  CommonVersionListTableProps
} from '@ar/pages/version-list/components/VersionListTable/VersionListTable'
import { VersionListColumnEnum } from '@ar/pages/version-list/components/VersionListTable/types'

const VersionListTableColumnConfig: CommonVersionListTableProps['columnConfigs'] = {
  [VersionListColumnEnum.Name]: { width: '100%' },
  [VersionListColumnEnum.Size]: { width: '100%' },
  [VersionListColumnEnum.DownloadCount]: { width: '100%' },
  [VersionListColumnEnum.LastModified]: { width: '100%' },
  [VersionListColumnEnum.PullCommand]: { width: '100%' },
  [VersionListColumnEnum.Actions]: { width: '10%' }
}
const DigestListTableColumnConfig: CommonVersionListTableProps['columnConfigs'] = {
  [VersionListColumnEnum.Digest]: { width: '100%' },
  [VersionListColumnEnum.Tags]: { width: '100%' },
  [VersionListColumnEnum.Size]: { width: '100%' },
  [VersionListColumnEnum.DownloadCount]: { width: '100%' },
  [VersionListColumnEnum.LastModified]: { width: '100%' },
  [VersionListColumnEnum.PullCommand]: { width: '100%' },
  [VersionListColumnEnum.Actions]: { width: '10%' }
}

function HelmVersionListTable(props: VersionListTableProps) {
  const { HAR_ENABLE_UNTAGGED_IMAGES_SUPPORT } = useFeatureFlags()
  const { parent } = useAppStore()
  if (HAR_ENABLE_UNTAGGED_IMAGES_SUPPORT || parent === Parent.OSS) {
    return <VersionListTable {...props} columnConfigs={DigestListTableColumnConfig} />
  }
  return <VersionListTable {...props} columnConfigs={VersionListTableColumnConfig} />
}

export default HelmVersionListTable
