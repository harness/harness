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

import React, { useContext } from 'react'
import type { IconName } from '@harnessio/icons'

import TreeNodeContent from '@ar/components/TreeView/TreeNodeContent'
import { TreeViewContext } from '@ar/components/TreeView/TreeViewContext'
import type { ArtifactTreeNodeViewProps } from '@ar/frameworks/Version/Version'

interface IArtifactTreeNode extends ArtifactTreeNodeViewProps {
  icon: IconName
  iconSize?: number
}

export default function ArtifactTreeNode(props: IArtifactTreeNode) {
  const { data, icon, iconSize = 24 } = props
  const { compact } = useContext(TreeViewContext)
  return (
    <TreeNodeContent
      icon={icon}
      iconSize={iconSize}
      label={data.name}
      downloads={data.downloadsCount}
      compact={compact}
    />
  )
}
