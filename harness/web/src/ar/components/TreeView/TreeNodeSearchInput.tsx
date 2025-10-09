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
import classNames from 'classnames'
import { ExpandingSearchInput, ExpandingSearchInputProps } from '@harnessio/uicore'
import { useStrings } from '@ar/frameworks/strings'
import TreeNode, { TreeNodeProps } from './TreeNode'

import type { ITreeNode } from './types'
import css from './TreeView.module.scss'

interface TreeNodeSearchInputProps extends ExpandingSearchInputProps {
  level?: number
  node: ITreeNode
  treeNodeProps?: Omit<TreeNodeProps, 'node' | 'heading'>
}

export default function TreeNodeSearchInput(props: TreeNodeSearchInputProps) {
  const { level = 0, treeNodeProps, className, node, ...rest } = props
  const { getString } = useStrings()
  return (
    <TreeNode
      className={classNames(className, css.stickyNode)}
      disabled
      node={node}
      level={level}
      onClick={e => e.stopPropagation()}
      heading={<ExpandingSearchInput alwaysExpanded placeholder={getString('search')} width="100%" {...rest} />}
      {...treeNodeProps}
    />
  )
}
