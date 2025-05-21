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

export enum NodeTypeEnum {
  File = 'File',
  Folder = 'Folder',
  Header = 'Header'
}

export enum TreeNodeTypeEnum {
  Loading = 'Loading',
  Empty = 'Empty',
  Error = 'Error',
  Node = 'Node',
  Search = 'Search',
  LoadMore = 'LoadMore',
  Header = 'Header'
}

export interface INode {
  id: string
  label: string
  value: string
  type: NodeTypeEnum
  disabled?: boolean
  metadata?: any
}

export interface ITreeNode extends INode {
  level: number
  treeNodeType: TreeNodeTypeEnum
  isLastChild?: boolean
  parentNode?: ITreeNode
}

export type INodeConfig<T = {}> = T & {
  searchTerm?: string
  page?: number
  hasMore?: string
  sort?: string
}

export interface IPaginationProps {
  page: number
  hasMore: boolean
}

export interface IFetchDataResult {
  data: Array<INode>
  pagination?: IPaginationProps
}
