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

import React, { PropsWithChildren, useContext } from 'react'
import type { IconName } from '@harnessio/icons'

import { useGetVersionDisplayName } from '@ar/hooks'
import type { RepositoryPackageType } from '@ar/common/types'
import TreeNodeContent from '@ar/components/TreeView/TreeNodeContent'
import { TreeViewContext } from '@ar/components/TreeView/TreeViewContext'
import type { VersionTreeNodeViewProps } from '@ar/frameworks/Version/Version'

interface IVersionTreeNode extends VersionTreeNodeViewProps {
  icon: IconName
  iconSize?: number
}

export default function VersionTreeNode(props: PropsWithChildren<IVersionTreeNode>) {
  const { data, icon, iconSize = 24 } = props
  const { compact } = useContext(TreeViewContext)
  const displayName = useGetVersionDisplayName(data.packageType as RepositoryPackageType, data.name)
  return (
    <TreeNodeContent
      icon={icon}
      iconSize={iconSize}
      label={displayName}
      size={data.size}
      downloads={data.downloadsCount}
      compact={compact}
    />
  )
}
