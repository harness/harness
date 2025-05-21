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
import { Virtuoso } from 'react-virtuoso'
import { Spinner } from '@blueprintjs/core'
import { Color, FontVariation } from '@harnessio/design-system'
import { DropDown, SelectOption, Text } from '@harnessio/uicore'

import { useDeepCompareEffect } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'

import TreeNode from './TreeNode'
import { TreeViewContext } from './TreeViewContext'
import TreeNodeSearchInput from './TreeNodeSearchInput'
import { IFetchDataResult, INodeConfig, ITreeNode, NodeTypeEnum, TreeNodeTypeEnum } from './types'
import {
  addElementAtIndex,
  addElementInSet,
  addElementsInArrayAtIndex,
  getComputedFilters,
  getEmptyTreeNodeConfig,
  getErrorTreeNodeConfig,
  getLoadingTreeNodeConfig,
  getLoadMoreTreeNodeConfig,
  getRootNodeTreeNodeConfig,
  getSearchNodeTreeNodeConfig,
  removeChildrenNodesFromArray,
  removeElementFromIndex,
  removeElementFromSet,
  removeNextSiblingNodesFromArray
} from './utils'
import { TreeNodeList } from './TreeNodeList'

interface TreeViewProps<T extends INodeConfig> {
  activePath: string
  setActivePath: (path: string) => void
  rootPath: string
  compact?: boolean
  fetchData: (node: ITreeNode, filters?: INodeConfig) => Promise<IFetchDataResult>
  onClick: (node: ITreeNode) => void
  renderNodeHeader?: (node: ITreeNode) => React.ReactNode
  renderNodeAction?: (node: ITreeNode) => React.ReactNode
  topItemCount?: number
  globalSearchConfig?: {
    className?: string
    searchTerm?: string
    sortOptions?: Array<SelectOption>
    sort?: string
  }
  globalFilters: Omit<T, keyof INodeConfig>
}

export default function TreeView<T>(props: TreeViewProps<T & INodeConfig>): JSX.Element {
  const {
    activePath,
    setActivePath,
    compact,
    fetchData,
    onClick,
    renderNodeHeader,
    renderNodeAction,
    topItemCount = 1
  } = props
  const [expandedIds, setExpandedIds] = React.useState<Set<string>>(new Set([]))
  const [loadingIds, setLoadingIds] = React.useState<Set<string>>(new Set([]))
  const [nodes, setNodes] = React.useState<Array<ITreeNode>>([])
  const isMounted = React.useRef(false)
  const [nodeConfig, setNodeConfig] = React.useState<Map<string, INodeConfig<T>>>(new Map())

  const { getString } = useStrings()

  // use this function after removing node from list.
  const clearAllTracesOfNodes = (ids: Set<string>) => {
    ids.forEach(id => {
      setExpandedIds(prev => removeElementFromSet(prev, id))
      setLoadingIds(prev => removeElementFromSet(prev, id))
    })
  }

  // use this function to update node config (page, filters, etc)
  const addOrUpdateNodeConfig = (node: ITreeNode, filters: INodeConfig = {}) => {
    setNodeConfig(prev => {
      const newMap = new Map(prev)
      const existingNodeConfig = newMap.get(node.id) || ({} as INodeConfig<T>)
      newMap.set(node.id, getComputedFilters(existingNodeConfig, filters))
      return newMap
    })
  }

  // base function to fetch nodes and transform to TreeNode
  const fetchNodes = async (node: ITreeNode, filters?: INodeConfig): Promise<Array<ITreeNode>> => {
    const level = node.level || 0
    return fetchData(node, filters)
      .then(result => {
        const { data, pagination } = result
        const childrenWithLevel: Array<ITreeNode> = data.map((child, idx) => ({
          ...child,
          level: level + 1,
          isLastChild: data.length - 1 === idx && !pagination?.hasMore,
          parentNode: node,
          disabled: child.disabled,
          treeNodeType: child.type === NodeTypeEnum.Header ? TreeNodeTypeEnum.Header : TreeNodeTypeEnum.Node
        }))
        if (childrenWithLevel.length === 0) {
          childrenWithLevel.push(getEmptyTreeNodeConfig(node))
        }
        if (pagination?.hasMore) {
          childrenWithLevel.push(getLoadMoreTreeNodeConfig(node, pagination.page))
        }
        addOrUpdateNodeConfig(node, filters)
        return childrenWithLevel
      })
      .catch((e: Error) => {
        const childrenWithLevel = [getErrorTreeNodeConfig(node, e)]
        return childrenWithLevel
      })
  }

  const fetchRootNodes = async (node: ITreeNode, index = 0, filters?: INodeConfig) => {
    const pathId = node.id
    setExpandedIds(prev => addElementInSet(prev, pathId))
    setLoadingIds(prev => addElementInSet(prev, pathId))
    // Add loading node in the tree
    setNodes(prev => {
      const updatedNodes = removeChildrenNodesFromArray(prev, index)
      clearAllTracesOfNodes(updatedNodes.removedIds)
      return addElementAtIndex(updatedNodes.result, index, getLoadingTreeNodeConfig(node))
    })
    return fetchNodes(node, filters).then(result => {
      setNodes(prev => {
        // Remove loading node in the tree
        const removedLoadingNodes = removeElementFromIndex(prev, 0)
        // clear all traces of removed loading node
        clearAllTracesOfNodes(removedLoadingNodes.removedIds)
        // add search node at the top
        const updatedNodes = [getSearchNodeTreeNodeConfig(node), ...removedLoadingNodes.result]
        // add new nodes to tree
        return addElementsInArrayAtIndex(updatedNodes, index, result)
      })
      setLoadingIds(prev => removeElementFromSet(prev, pathId))
    })
  }

  const fetchTreeNodes = async (node: ITreeNode, index = 0, filters: INodeConfig = {}) => {
    const pathId = node.id
    setExpandedIds(prev => addElementInSet(prev, pathId))
    setLoadingIds(prev => addElementInSet(prev, pathId))
    // add loading node in the tree
    setNodes(prev => {
      const updatedNodes = removeChildrenNodesFromArray(prev, index)
      clearAllTracesOfNodes(updatedNodes.removedIds)
      return addElementAtIndex(updatedNodes.result, index, getLoadingTreeNodeConfig(node))
    })
    return fetchNodes(node, filters).then(result => {
      setNodes(prev => {
        // remove loading node in the tree
        const updatedNodes = removeChildrenNodesFromArray(prev, index)
        // clear all traces of removed loading node
        clearAllTracesOfNodes(updatedNodes.removedIds)
        // add element in tree
        return addElementsInArrayAtIndex(updatedNodes.result, index, result)
      })
      setLoadingIds(prev => removeElementFromSet(prev, pathId))
    })
  }

  const fetchNextPage = async (node: ITreeNode, index: number) => {
    const { parentNode } = node
    if (!parentNode) return
    const filters = nodeConfig.get(parentNode.id) || ({} as INodeConfig)
    const currentPage = filters.page ?? 0
    // add loading node in tree
    setNodes(prev => {
      return addElementAtIndex(prev, index, getLoadingTreeNodeConfig(node))
    })
    return fetchNodes(parentNode, getComputedFilters(filters, { page: currentPage + 1 })).then(result => {
      setNodes(prev => {
        // remove loading node
        const removedLoadingNodes = removeElementFromIndex(prev, index + 1)
        clearAllTracesOfNodes(removedLoadingNodes.removedIds)
        // add new nodes
        const addedChildrenNodes = addElementsInArrayAtIndex(removedLoadingNodes.result, index, result)
        // remove load more node
        const removedLoadMoreNode = removeElementFromIndex(addedChildrenNodes, index)
        clearAllTracesOfNodes(removedLoadMoreNode.removedIds)
        return removedLoadMoreNode.result
      })
    })
  }

  const handleCollapseTreeNode = (node: ITreeNode, index: number) => {
    const pathId = node.id
    setExpandedIds(prev => removeElementFromSet(prev, pathId))
    setNodes(prev => {
      // remove all children nodes from tree
      const removedChildrenNodes = removeChildrenNodesFromArray(prev, index)
      // clear all traces of removed children nodes
      clearAllTracesOfNodes(removedChildrenNodes.removedIds)
      return removedChildrenNodes.result
    })
  }

  const handleClickTreeNode = (node: ITreeNode, index: number, isInitialising?: boolean) => {
    const pathId = node.id
    // do not do any thing if node type is file
    if (node.type === NodeTypeEnum.File) return
    // do not do anything if already expanded and its initialising because of virtual list scroll
    if (expandedIds.has(pathId) && isInitialising) return
    // collapse if already expanded
    if (expandedIds.has(pathId)) {
      handleCollapseTreeNode(node, index)
      return
    }
    // do not do anything if already loading node details
    if (loadingIds.has(pathId)) return
    // expand if not expanded
    fetchTreeNodes(node, index)
  }

  const handleSearchTreeNode = async (node: ITreeNode, index: number, filters: INodeConfig) => {
    const pathId = node.id
    if (!node.parentNode) return
    const existingNodeConfig = nodeConfig.get(node.parentNode.id) || {}
    // remove all children nodes and add loading node in tree
    setNodes(prev => {
      // remove all siblings nodes
      const updatedNodes = removeNextSiblingNodesFromArray(prev, index)
      // clear all traces of removed nodes
      clearAllTracesOfNodes(updatedNodes.removedIds)
      // add loading node
      return addElementAtIndex(updatedNodes.result, index, getLoadingTreeNodeConfig(node))
    })
    return fetchNodes(node.parentNode, getComputedFilters(existingNodeConfig, filters)).then(result => {
      setNodes(prev => {
        // remove loading node
        const updatedNodes = removeNextSiblingNodesFromArray(prev, index)
        // clear all traces of removed loading node
        clearAllTracesOfNodes(updatedNodes.removedIds)
        // add new nodes
        return addElementsInArrayAtIndex(updatedNodes.result, index, result)
      })
      setLoadingIds(prev => addElementInSet(prev, pathId))
    })
  }

  useDeepCompareEffect(() => {
    if (!isMounted.current) {
      isMounted.current = true
      const rootNodeConfig = getRootNodeTreeNodeConfig(props.rootPath)
      const rootNodeIndex = 0
      const rootNodeFilters = getComputedFilters(props.globalFilters, {
        searchTerm: props.globalSearchConfig?.searchTerm,
        sort: props.globalSearchConfig?.sort
      })
      fetchRootNodes(rootNodeConfig, rootNodeIndex, rootNodeFilters)
    } else {
      const rootSearchNode = nodes[0]
      const rootNodeIndex = 0
      const rootNodeFilters = getComputedFilters(props.globalFilters, { page: 0 })
      handleSearchTreeNode(rootSearchNode, rootNodeIndex, rootNodeFilters)
    }
  }, [props.globalFilters, props.rootPath])

  return (
    <TreeViewContext.Provider value={{ activePath, setActivePath, compact, rootPath: props.rootPath }}>
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
                  defaultValue={node.parentNode?.id && nodeConfig.get(node.parentNode.id)?.searchTerm}
                  onChange={val => handleSearchTreeNode(node, index, { searchTerm: val, page: 0 })}
                  node={node}
                  treeNodeProps={{
                    alwaysShowAction: true,
                    actionElement: props.globalSearchConfig?.sortOptions && (
                      <DropDown
                        icon="main-sort"
                        items={props.globalSearchConfig.sortOptions}
                        value={node.parentNode?.id && nodeConfig.get(node.parentNode.id)?.sort}
                        onChange={option => {
                          handleSearchTreeNode(node, index, { sort: option.value as string, page: 0 })
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
                  isOpen={activePath.startsWith(node.id)}
                  isActive={activePath === node.id}
                  isLastChild={node.isLastChild}
                  actionElement={renderNodeAction ? renderNodeAction(node) : undefined}
                  compact={compact}
                  onNodeClick={isInitialising => {
                    if (!isInitialising) onClick(node)
                    handleClickTreeNode(node, index, isInitialising)
                  }}
                />
              )
          }
        }}
      />
    </TreeViewContext.Provider>
  )
}
