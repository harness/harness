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

import css from './TreeView.module.scss'

export const TreeNodeList = React.forwardRef<HTMLDivElement, React.HTMLProps<HTMLDivElement>>(
  ({ style, children, className, ...rest }, ref) => {
    const handleKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
      const items = Array.from(document.querySelectorAll('[data-tree-item]')) as HTMLElement[]
      const idx = items.findIndex(el => el === document.activeElement)
      if (idx === -1) return
      const activeElement = items[idx]
      const isDirectory = activeElement.getAttribute('data-is-directory') === 'true'
      const isOpen = activeElement.getAttribute('data-is-expanded') === 'true'

      switch (e.key) {
        case 'ArrowDown': {
          e.preventDefault()
          const next = items[idx + 1]
          next?.focus()
          return
        }
        case 'ArrowUp': {
          e.preventDefault()
          const prev = items[idx - 1]
          prev?.focus()
          return
        }
        case 'ArrowRight': {
          e.preventDefault()
          if (!isDirectory || isOpen) return
          e.currentTarget.focus()
          activeElement.click()
          return
        }
        case 'ArrowLeft': {
          e.preventDefault()
          if (!isDirectory || !isOpen) return
          e.currentTarget.focus()
          activeElement.click()
          return
        }
        case 'Enter':
          e.preventDefault()
          e.currentTarget.focus()
          activeElement.click()
          return
        default:
          return
      }
    }
    return (
      <div
        {...rest}
        style={style}
        ref={ref}
        className={classNames(className, css.treeList)}
        role="tree"
        onKeyDown={handleKeyDown}>
        {children}
      </div>
    )
  }
)

TreeNodeList.displayName = 'TreeNodeList'
