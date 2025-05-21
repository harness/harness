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

import React, { useContext, useEffect, useMemo, useRef, useState, type PropsWithChildren } from 'react'
import classNames from 'classnames'
import { Icon } from '@harnessio/icons'
import { Container } from '@harnessio/uicore'

import childImage from './images/child.svg?url'
import lastChildImage from './images/last-child.svg?url'
import lineImage from './images/line.svg?url'

import { ITreeNode, NodeTypeEnum } from './types'
import { TreeViewContext } from './TreeViewContext'

import css from './TreeView.module.scss'

export interface TreeNodeProps extends React.HTMLAttributes<HTMLDivElement> {
  heading: string | React.ReactNode
  node: ITreeNode
  isActive?: boolean
  isOpen?: boolean
  nodeType?: NodeTypeEnum
  level?: number
  compact?: boolean
  disabled?: boolean
  onNodeClick?: (isInitialising?: boolean) => void
  actionElement?: React.ReactNode
  alwaysShowAction?: boolean
  isLastChild?: boolean
}

export default function TreeNode(props: PropsWithChildren<TreeNodeProps>) {
  const {
    isOpen,
    nodeType = NodeTypeEnum.File,
    level = 0,
    onNodeClick,
    compact = true,
    disabled,
    isActive,
    actionElement,
    alwaysShowAction,
    isLastChild = false,
    className,
    heading,
    node,
    ...rest
  } = props
  const ref = useRef<HTMLDivElement>(null)
  const [open, setOpen] = useState(isOpen)
  const { rootPath } = useContext(TreeViewContext)

  const getParentNodeList = (treeNode: ITreeNode, result: Array<ITreeNode> = []) => {
    if (treeNode.parentNode && treeNode.parentNode.id !== rootPath) {
      result.push(treeNode.parentNode)
      getParentNodeList(treeNode.parentNode, result)
    }
    return result
  }

  const parentNodeLevels = useMemo(() => getParentNodeList(node, []).reverse(), [])

  const handleClickNode = () => {
    if (disabled) return
    if (nodeType === NodeTypeEnum.Folder) setOpen(!open)
    onNodeClick?.()
  }

  useEffect(() => {
    if (open && ref.current) {
      ref.current.focus()
      onNodeClick?.(true)
      if (isActive) {
        ref.current.scrollIntoView({ behavior: 'smooth', block: 'center', inline: 'center' })
      }
    }
  }, [])

  return (
    <div
      data-level={level}
      data-id={node.id}
      data-parent-id={node.parentNode?.id}
      data-last-child={isLastChild}
      data-is-directory={nodeType === NodeTypeEnum.Folder}
      data-is-expanded={open}
      data-tree-item
      ref={ref}
      className={classNames(css.treeNode, className, {
        [css.active]: isActive,
        [css.disabled]: disabled
      })}
      onClick={e => {
        e.currentTarget.focus()
        handleClickNode()
      }}
      tabIndex={0}
      {...rest}>
      <Container className={classNames(css.header)}>
        {parentNodeLevels.slice(1).map((_each, indx) => {
          const img = _each.isLastChild ? undefined : lineImage
          return <div key={indx} className={css.levelImg} style={{ backgroundImage: `url(${img})` }}></div>
        })}
        {level > 0 && (
          <div
            className={css.levelImg}
            style={{ backgroundImage: `url(${isLastChild ? lastChildImage : childImage})` }}></div>
        )}

        {nodeType === NodeTypeEnum.Folder && <Icon name={open ? 'chevron-down' : 'chevron-right'} />}
        <Container padding={compact ? 'xsmall' : 'small'} className={css.headingContent}>
          {props.heading}
        </Container>
        {actionElement && (alwaysShowAction || isActive) && <Container>{actionElement}</Container>}
      </Container>
      {open && <Container>{props.children}</Container>}
    </div>
  )
}
