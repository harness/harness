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

import React, { useEffect, useRef, useState, type PropsWithChildren } from 'react'
import classNames from 'classnames'
import { Icon } from '@harnessio/icons'
import { Container } from '@harnessio/uicore'

import childImage from './images/child.svg?url'
import lastChildImage from './images/last-child.svg?url'
import lineImage from './images/line.svg?url'

import type { NodeSpec } from './TreeViewContext'

import css from './TreeView.module.scss'

export enum NodeTypeEnum {
  File = 'File',
  Folder = 'Folder'
}

export interface TreeNodeProps<T = unknown> extends React.HTMLAttributes<HTMLLIElement> {
  heading: string | React.ReactNode
  isActive?: boolean
  isOpen?: boolean
  nodeType?: NodeTypeEnum
  level?: number
  compact?: boolean
  disabled?: boolean
  onClick?: () => void
  actionElement?: React.ReactNode
  alwaysShowAction?: boolean
  isLastChild?: boolean
  parentNodeLevels?: Array<NodeSpec<T>>
}

export default function TreeNode<T>(props: PropsWithChildren<TreeNodeProps<T>>) {
  const {
    isOpen,
    nodeType = NodeTypeEnum.File,
    level = 0,
    onClick,
    compact = true,
    disabled,
    isActive,
    actionElement,
    alwaysShowAction,
    parentNodeLevels = [],
    isLastChild = false,
    className,
    ...rest
  } = props
  const ref = useRef<HTMLLIElement>(null)
  const [open, setOpen] = useState(isOpen)
  const [isMounted, setIsMounted] = useState(false)

  useEffect(() => {
    if (open && ref.current) {
      ref.current.scrollIntoView({ behavior: 'smooth', block: 'center', inline: 'center' })
    }
  }, [])

  return (
    <li
      data-level={2}
      data-last-child={isLastChild}
      ref={ref}
      className={classNames(css.treeNode, className)}
      {...rest}>
      <Container
        tabIndex={0}
        onMouseEnter={() => setIsMounted(true)}
        onMouseLeave={() => setIsMounted(false)}
        className={classNames(css.header, {
          [css.active]: isActive,
          [css.disabled]: disabled
        })}
        onClick={() => {
          if (disabled) return
          if (nodeType === NodeTypeEnum.Folder) setOpen(!open)
          onClick?.()
        }}>
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
        {actionElement && (isMounted || alwaysShowAction || isActive) && <Container>{actionElement}</Container>}
      </Container>
      {open && <Container>{props.children}</Container>}
    </li>
  )
}
