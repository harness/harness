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

import { ITreeNode, NodeTypeEnum, TreeNodeTypeEnum } from './types'

export function addElementInSet<T>(prev: Set<T>, id: T): Set<T> {
  const newSet = new Set(prev)
  newSet.add(id)
  return newSet
}

export function removeElementFromSet<T>(prev: Set<T>, id: T) {
  const newSet = new Set(prev)
  newSet.delete(id)
  return newSet
}

export function addElementsInArrayAtIndex<T>(arr: Array<T>, index: number, childrenNodes: Array<T>): Array<T> {
  const newItems = [...arr]
  newItems.splice(index + 1, 0, ...childrenNodes)
  return newItems
}

export function addElementAtIndex<T>(arr: Array<T>, index: number, node: T): Array<T> {
  const newItems = [...arr]
  newItems.splice(index + 1, 0, node)
  return newItems
}

export function removeElementFromIndex<T>(
  arr: Array<T & { level: number; id: string }>,
  index: number
): { result: Array<T>; removedIds: Set<string> } {
  const result = [...arr]
  const removedIds = new Set<string>()
  result.splice(index, 1)
  removedIds.add(arr[index].id)
  return { result, removedIds }
}

export function removeChildrenNodesFromArray<T>(
  arr: Array<T & { level: number; id: string }>,
  index: number
): { result: Array<T>; removedIds: Set<string> } {
  const newItems = [...arr]
  const currentNode = newItems[index]
  if (!currentNode) return { result: newItems, removedIds: new Set() }
  const currentLevel = currentNode.level
  const removedIds = new Set<string>()
  let i = index + 1
  while (i < newItems.length && newItems[i].level > currentLevel) {
    removedIds.add(newItems[i].id)
    i++
  }
  newItems.splice(index + 1, i - index - 1)
  return { result: newItems, removedIds }
}

export function removeNextSiblingNodesFromArray<T>(
  arr: Array<T & { level: number; id: string }>,
  index: number
): { result: Array<T>; removedIds: Set<string> } {
  const newItems = [...arr]
  const currentNode = newItems[index]
  if (!currentNode) return { result: newItems, removedIds: new Set() }
  const currentLevel = currentNode.level
  const removedIds = new Set<string>()
  let i = index + 1
  while (i < newItems.length && newItems[i].level >= currentLevel) {
    removedIds.add(newItems[i].id)
    i++
  }
  newItems.splice(index + 1, i - index - 1)
  return { result: newItems, removedIds }
}

export function getComputedFilters<T, U>(existingFilters: T, newFilters: U): T & U {
  const filters = { ...existingFilters, ...newFilters }
  return filters
}

export const getEmptyTreeNodeConfig = (node: ITreeNode) => {
  const pathId = node.id
  const level = node.level || 0
  return {
    id: `${pathId}/empty`,
    label: '',
    value: '',
    type: NodeTypeEnum.File,
    level: level + 1,
    treeNodeType: TreeNodeTypeEnum.Empty,
    isLastChild: true,
    parentNode: node,
    metadata: {}
  }
}

export const getErrorTreeNodeConfig = (node: ITreeNode, err: Error) => {
  const pathId = node.id
  const level = node.level || 0
  return {
    id: `${pathId}/error`,
    label: '',
    value: '',
    type: NodeTypeEnum.File,
    level: level + 1,
    treeNodeType: TreeNodeTypeEnum.Error,
    isLastChild: true,
    parentNode: node,
    metadata: err
  }
}

export const getLoadingTreeNodeConfig = (node: ITreeNode) => {
  const pathId = node.id
  const level = node.level || 0
  return {
    id: `${pathId}/loading`,
    label: '',
    value: '',
    type: NodeTypeEnum.File,
    level: level + 1,
    treeNodeType: TreeNodeTypeEnum.Loading,
    isLastChild: true,
    parentNode: node
  }
}

export const getLoadMoreTreeNodeConfig = (node: ITreeNode, page: number) => {
  const pathId = node.id
  return {
    id: `${pathId}/loadMore/${page}`,
    label: '',
    value: '',
    type: NodeTypeEnum.File,
    level: node.level + 1,
    treeNodeType: TreeNodeTypeEnum.LoadMore,
    isLastChild: true,
    parentNode: node
  }
}

export const getSearchNodeTreeNodeConfig = (node: ITreeNode) => {
  const pathId = node.id
  return {
    id: `${pathId}/search`,
    label: '',
    value: '',
    type: NodeTypeEnum.File,
    level: node.level + 1,
    treeNodeType: TreeNodeTypeEnum.Search,
    isLastChild: true,
    parentNode: node
  }
}

export const getRootNodeTreeNodeConfig = (rootPath: string) => {
  return {
    id: rootPath,
    level: -1,
    label: rootPath,
    value: rootPath,
    type: NodeTypeEnum.Folder,
    metadata: {},
    treeNodeType: TreeNodeTypeEnum.Node
  }
}
