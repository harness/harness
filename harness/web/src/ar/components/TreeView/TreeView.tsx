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

import React, { useEffect } from 'react'
import { Virtuoso } from 'react-virtuoso'
import { Spinner } from '@blueprintjs/core'
import { Color, FontVariation } from '@harnessio/design-system'
import { DropDown, SelectOption, Text } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'

import TreeNode, { NodeStateEnum } from './TreeNode'
import { TreeViewContext } from './TreeViewContext'
import { INode, INodeConfig, ITreeNode, NodeTypeEnum, TreeNodeTypeEnum } from './types'
import {
  addElementAtIndex,
  addElementInSet,
  addElementsInArrayAtIndex,
  getEmptyTreeNodeConfig,
  getErrorTreeNodeConfig,
  getLoadingTreeNodeConfig,
  getRootNodeTreeNodeConfig,
  getSearchNodeTreeNodeConfig,
  removeChildrenNodesFromArray,
  removeElementFromIndex,
  removeElementFromSet
} from './utils'
import { TreeNodeList } from './TreeNodeList'
import TreeNodeSearchInput from './TreeNodeSearchInput'

interface TreeViewProps {
  activePath: string
  setActivePath: (path: string) => void
  rootNodes: Array<INode>
  loadingRootNodes: boolean
  rootPath: string
  compact?: boolean
  fetchData: (node: ITreeNode, filters?: INodeConfig) => Promise<Array<INode>>
  onClick: (node: ITreeNode) => void
  renderNodeHeader?: (node: ITreeNode) => React.ReactNode
  renderNodeAction?: (node: ITreeNode) => React.ReactNode
  topItemCount?: number
  initialised?: boolean
  globalSearchConfig?: {
    className?: string
    searchTerm?: string
    sort?: string
    sortOptions?: Array<SelectOption>
    onChange?: (searchTerm: string, sort?: string) => void
  }
}

export default function TreeView(props: TreeViewProps): JSX.Element {
  const {
    activePath,
    setActivePath,
    compact,
    fetchData,
    onClick,
    renderNodeHeader,
    renderNodeAction,
    topItemCount = 1,
    rootNodes,
    loadingRootNodes,
    initialised
  } = props
  const [expandedIds, setExpandedIds] = React.useState<Set<string>>(new Set([]))
  const [contentLoadedIds, setContentLoadedIds] = React.useState<Set<string>>(new Set([]))
  const [loadingIds, setLoadingIds] = React.useState<Set<string>>(new Set([]))
  const [nodes, setNodes] = React.useState<Array<ITreeNode>>([])

  const { getString } = useStrings()

  // use this function after removing node from list.
  const clearAllTracesOfNodes = (ids: Set<string>) => {
    ids.forEach(id => {
      setExpandedIds(prev => removeElementFromSet(prev, id))
      setLoadingIds(prev => removeElementFromSet(prev, id))
      setContentLoadedIds(prev => removeElementFromSet(prev, id))
    })
  }

  const mapNodeTypeToTreeNodeType = (nodeType: NodeTypeEnum): TreeNodeTypeEnum => {
    switch (nodeType) {
      case NodeTypeEnum.Header:
        return TreeNodeTypeEnum.Header
      case NodeTypeEnum.LoadMore:
        return TreeNodeTypeEnum.LoadMore
      case NodeTypeEnum.Error:
        return TreeNodeTypeEnum.Error
      case NodeTypeEnum.File:
      case NodeTypeEnum.Folder:
      default:
        return TreeNodeTypeEnum.Node
    }
  }

  const mapNodesToTreeNodes = (parentNode: ITreeNode, nodeList: Array<INode>): Array<ITreeNode> => {
    const parentNodeLevel = parentNode.level ?? -1
    const treeNodeList: Array<ITreeNode> = nodeList.map((node, idx) => ({
      ...node,
      level: parentNodeLevel + 1,
      isLastChild: idx === nodeList.length - 1,
      parentNode: parentNode,
      disabled: node.disabled,
      treeNodeType: mapNodeTypeToTreeNodeType(node.type)
    }))
    if (treeNodeList.length === 0) {
      treeNodeList.push(getEmptyTreeNodeConfig(parentNode))
    }
    return treeNodeList
  }

  // base function to fetch nodes and transform to TreeNode
  const fetchNodes = async (node: ITreeNode): Promise<Array<ITreeNode>> => {
    return fetchData(node)
      .then(result => {
        const treeNodes = mapNodesToTreeNodes(node, result)
        return treeNodes
      })
      .catch((e: Error) => {
        const treeNodes = [getErrorTreeNodeConfig(node, e)]
        return treeNodes
      })
  }

  const fetchTreeNodes = async (node: ITreeNode, index = 0) => {
    const pathId = node.id
    setExpandedIds(prev => addElementInSet(prev, pathId))
    setLoadingIds(prev => addElementInSet(prev, pathId))
    // add loading node in the tree
    setNodes(prev => {
      const updatedNodes = removeChildrenNodesFromArray(prev, index)
      clearAllTracesOfNodes(updatedNodes.removedIds)
      return addElementAtIndex(updatedNodes.result, index, getLoadingTreeNodeConfig(node))
    })
    return fetchNodes(node).then(result => {
      setNodes(prev => {
        // remove loading node in the tree
        const updatedNodes = removeChildrenNodesFromArray(prev, index)
        // clear all traces of removed loading node
        clearAllTracesOfNodes(updatedNodes.removedIds)
        // add element in tree
        return addElementsInArrayAtIndex(updatedNodes.result, index, result)
      })
      setLoadingIds(prev => removeElementFromSet(prev, pathId))
      setContentLoadedIds(prev => addElementInSet(prev, pathId))
    })
  }

  const fetchNextPage = async (node: ITreeNode, index: number) => {
    const { parentNode, level } = node
    if (!parentNode) return
    // add loading node in tree
    setNodes(prev => {
      return addElementAtIndex(prev, index, getLoadingTreeNodeConfig(node))
    })
    return fetchNodes(node).then(result => {
      const transformedTreeNodesForNextPage = result.map(each => ({
        ...each,
        level,
        parentNode
      }))
      setNodes(prev => {
        // remove loading node
        const removedLoadingNodes = removeElementFromIndex(prev, index + 1)
        clearAllTracesOfNodes(removedLoadingNodes.removedIds)
        // add new nodes
        const addedChildrenNodes = addElementsInArrayAtIndex(
          removedLoadingNodes.result,
          index,
          transformedTreeNodesForNextPage
        )
        // remove load more node
        const removedLoadMoreNode = removeElementFromIndex(addedChildrenNodes, index)
        clearAllTracesOfNodes(removedLoadMoreNode.removedIds)
        return removedLoadMoreNode.result
      })
    })
  }

  const handleCollapseTreeNode = (node: ITreeNode, index: number) => {
    const pathId = node.id
    setNodes(prev => {
      // remove all children nodes from tree
      const removedChildrenNodes = removeChildrenNodesFromArray(prev, index)
      // clear all traces of removed children nodes
      removedChildrenNodes.removedIds.add(pathId)
      clearAllTracesOfNodes(removedChildrenNodes.removedIds)
      return removedChildrenNodes.result
    })
  }

  const handleClickTreeNode = (node: ITreeNode, index: number, nextState: NodeStateEnum) => {
    const pathId = node.id
    // do not do any thing if node type is file
    if (node.type === NodeTypeEnum.File) return
    // do not do anything if already loaded content and its initialising because of virtual list scroll
    if (contentLoadedIds.has(pathId) && nextState === NodeStateEnum.EXPANDED) return
    // collapse if already expanded & not initialising
    if (nextState === NodeStateEnum.COLLAPSED) {
      handleCollapseTreeNode(node, index)
      return
    }
    // do not do anything if already loading node details
    if (loadingIds.has(pathId)) return
    // expand if not expanded
    fetchTreeNodes(node, index)
  }

  // set initial root nodes
  useEffect(() => {
    const parentNodeForRootNode = getRootNodeTreeNodeConfig(props.rootPath)
    const initialNodes = []
    if (props.globalSearchConfig) {
      initialNodes.push(getSearchNodeTreeNodeConfig(parentNodeForRootNode))
    }
    if (!loadingRootNodes) {
      initialNodes.push(...mapNodesToTreeNodes(parentNodeForRootNode, rootNodes))
    } else {
      initialNodes.push(...[getLoadingTreeNodeConfig(parentNodeForRootNode)])
    }
    setNodes(initialNodes)
    setContentLoadedIds(new Set([]))
  }, [loadingRootNodes])

  // set initial expanded ids based on active path
  useEffect(() => {
    if (!initialised) return
    const initialExpandedIds = new Set<string>([])
    let prevPath = ''
    activePath.split('/').forEach(each => {
      const newPath = prevPath ? `${prevPath}/${each}` : each
      initialExpandedIds.add(newPath)
      prevPath = newPath
    })
    setExpandedIds(initialExpandedIds)
  }, [initialised])

  if (!initialised) return <Spinner size={Spinner.SIZE_SMALL} />

  return (
    <TreeViewContext.Provider
      value={{
        activePath,
        setActivePath,
        compact,
        rootPath: props.rootPath,
        contentLoadedIds
      }}>
      <Virtuoso
        data={nodes}
        components={{
          List: TreeNodeList
        }}
        topItemCount={topItemCount}
        itemContent={(index, node) => {
          switch (node.treeNodeType) {
            case TreeNodeTypeEnum.Loading:
              return (
                <TreeNode
                  key={node.id}
                  node={node}
                  heading={<Spinner size={Spinner.SIZE_SMALL} />}
                  nodeType={NodeTypeEnum.File}
                  disabled
                  isLastChild={node.isLastChild}
                  level={node.level}
                />
              )
            case TreeNodeTypeEnum.Error:
              return (
                <TreeNode
                  key={node.id}
                  node={node}
                  heading={
                    <Text
                      icon="error"
                      lineClamp={1}
                      color={Color.ERROR}
                      iconProps={{ color: Color.ERROR }}
                      font={{ variation: FontVariation.BODY }}>
                      {node.metadata}
                    </Text>
                  }
                  nodeType={NodeTypeEnum.File}
                  disabled
                  isLastChild={node.isLastChild}
                  level={node.level}
                />
              )
            case TreeNodeTypeEnum.Empty:
              return (
                <TreeNode
                  key={node.id}
                  node={node}
                  disabled
                  heading={<Text>{getString('noResultsFound')}</Text>}
                  isLastChild={node.isLastChild}
                  nodeType={NodeTypeEnum.File}
                  level={node.level}
                />
              )
            case TreeNodeTypeEnum.LoadMore:
              return (
                <TreeNode
                  key={node.id}
                  node={node}
                  heading={<Text color={Color.PRIMARY_7}>{getString('loadMore')}</Text>}
                  isLastChild={node.isLastChild}
                  nodeType={NodeTypeEnum.File}
                  level={node.level}
                  onNodeClick={() => {
                    fetchNextPage(node, index)
                  }}
                />
              )
            case TreeNodeTypeEnum.Header:
              return (
                <TreeNode
                  key={node.id}
                  node={node}
                  heading={
                    <Text font={{ weight: 'bold', variation: FontVariation.BODY }} color={Color.GREY_800}>
                      {node.label}
                    </Text>
                  }
                  disabled
                  isLastChild={false}
                  nodeType={NodeTypeEnum.File}
                  level={node.level}
                />
              )
            case TreeNodeTypeEnum.Search:
              return (
                <TreeNodeSearchInput
                  className={props.globalSearchConfig?.className}
                  key={node.id}
                  level={node.level}
                  defaultValue={props.globalSearchConfig?.searchTerm ?? ''}
                  onChange={val => props.globalSearchConfig?.onChange?.(val, props.globalSearchConfig?.sort)}
                  node={node}
                  treeNodeProps={{
                    alwaysShowAction: true,
                    actionElement: props.globalSearchConfig?.sortOptions && (
                      <DropDown
                        icon="main-sort"
                        items={props.globalSearchConfig.sortOptions}
                        value={props.globalSearchConfig?.sort}
                        onChange={option => {
                          props.globalSearchConfig?.onChange?.(
                            props.globalSearchConfig?.searchTerm ?? '',
                            option.value as string
                          )
                        }}
                        usePortal
                      />
                    )
                  }}
                />
              )
            case TreeNodeTypeEnum.Node:
            default:
              return (
                <TreeNode
                  key={node.id}
                  heading={
                    renderNodeHeader ? (
                      renderNodeHeader(node)
                    ) : (
                      <Text lineClamp={1} font={{ variation: FontVariation.BODY }} color={Color.GREY_800}>
                        {node.label}
                      </Text>
                    )
                  }
                  node={node}
                  nodeType={node.type}
                  level={node.level}
                  isOpen={expandedIds.has(node.id)}
                  initialised={contentLoadedIds.has(node.id)}
                  isActive={activePath === node.id}
                  isLastChild={node.isLastChild}
                  actionElement={renderNodeAction ? renderNodeAction(node) : undefined}
                  compact={compact}
                  onChangeState={nextState => {
                    handleClickTreeNode(node, index, nextState)
                  }}
                  onNodeClick={() => {
                    onClick(node)
                  }}
                />
              )
          }
        }}
      />
    </TreeViewContext.Provider>
  )
}
