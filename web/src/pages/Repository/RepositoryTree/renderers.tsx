/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react'
import cx from 'classnames'
import type { TreeRenderProps } from 'react-complex-tree'
import { Button, Classes, Collapse, Colors, Icon, InputGroup } from '@blueprintjs/core'

export const renderers: TreeRenderProps = {
  renderTreeContainer: props => (
    <div className={cx(Classes.TREE)}>
      <ul className={cx(Classes.TREE_ROOT, Classes.TREE_NODE_LIST)} {...props.containerProps}>
        {props.children}
      </ul>
    </div>
  ),

  renderItemsContainer: props => (
    <ul className={cx(Classes.TREE_NODE_LIST)} {...props.containerProps}>
      {props.children}
    </ul>
  ),

  renderItem: props => (
    <li
      className={cx(
        Classes.TREE_NODE,
        (props.context.isSelected || props.context.isDraggingOver) && Classes.TREE_NODE_SELECTED
      )}
      {...(props.context.itemContainerWithChildrenProps as Unknown)}>
      <div
        className={cx(Classes.TREE_NODE_CONTENT, `${Classes.TREE_NODE_CONTENT}-${props.depth}`)}
        {...(props.context.itemContainerWithoutChildrenProps as Unknown)}
        {...(props.context.interactiveElementProps as Unknown)}>
        {props.item.hasChildren ? props.arrow : <span className={Classes.TREE_NODE_CARET_NONE} />}
        {props.item.data.icon !== undefined ? (
          props.item.data.icon === null ? null : (
            <Icon icon={props.item.data.icon} className={Classes.TREE_NODE_ICON} />
          )
        ) : (
          (() => {
            const icon = !props.item.hasChildren
              ? 'document'
              : props.context.isExpanded
              ? 'folder-open'
              : 'folder-close'
            return <Icon icon={icon} className={Classes.TREE_NODE_ICON} />
          })()
        )}
        {props.title}
      </div>
      <div
        className={cx(Classes.COLLAPSE)}
        style={
          props.context.isExpanded
            ? {
                height: 'auto',
                overflowY: 'visible',
                transition: 'none 0s ease 0s'
              }
            : {}
        }>
        <Collapse isOpen={props.context.isExpanded} transitionDuration={0}>
          {props.children}
        </Collapse>
      </div>
    </li>
  ),

  renderItemArrow: props => (
    <Icon
      icon="chevron-right"
      className={cx(
        Classes.TREE_NODE_CARET,
        props.context.isExpanded ? Classes.TREE_NODE_CARET_OPEN : Classes.TREE_NODE_CARET_CLOSED
      )}
      {...(props.context.arrowProps as Unknown)}
    />
  ),

  renderItemTitle: ({ title, context, info }) => {
    if (!info.isSearching || !context.isSearchMatching) {
      return <span className={Classes.TREE_NODE_LABEL}>{title}</span>
    } else {
      const startIndex = title.toLowerCase().indexOf((info.search as string).toLowerCase())
      return (
        <React.Fragment>
          {startIndex > 0 && <span>{title.slice(0, startIndex)}</span>}
          <span className="rct-tree-item-search-highlight">
            {title.slice(startIndex, startIndex + (info.search as string).length)}
          </span>
          {startIndex + (info.search as string).length < title.length && (
            <span>{title.slice(startIndex + (info.search as string).length, title.length)}</span>
          )}
        </React.Fragment>
      )
    }
  },

  renderDragBetweenLine: ({ draggingPosition, lineProps }) => (
    <div
      {...lineProps}
      style={{
        position: 'absolute',
        right: '0',
        top:
          draggingPosition.targetType === 'between-items' && draggingPosition.linePosition === 'top'
            ? '0px'
            : draggingPosition.targetType === 'between-items' && draggingPosition.linePosition === 'bottom'
            ? '-4px'
            : '-2px',
        left: `${draggingPosition.depth * 23}px`,
        height: '4px',
        backgroundColor: Colors.BLUE3
      }}
    />
  ),

  renderRenameInput: props => (
    <form {...props.formProps} style={{ display: 'contents' }}>
      <span className={Classes.TREE_NODE_LABEL}>
        <input {...props.inputProps} ref={props.inputRef} className="rct-tree-item-renaming-input" />
      </span>
      <span className={Classes.TREE_NODE_SECONDARY_LABEL}>
        <Button icon="tick" {...(props.submitButtonProps as Unknown)} type="submit" minimal={true} small={true} />
      </span>
    </form>
  ),

  renderSearchInput: props => (
    <div className={cx('rct-tree-search-input-container')}>
      <InputGroup {...(props.inputProps as Unknown)} placeholder="Search..." />
    </div>
  ),

  renderDepthOffset: 1
}
