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
import { defaultTo, omit } from 'lodash-es'
import { useHistory } from 'react-router-dom'
import type { IconName } from '@harnessio/icons'

import { useParentHooks, useRoutes } from '@ar/hooks'
import TreeNode, { NodeTypeEnum } from '@ar/components/TreeView/TreeNode'
import TreeNodeContent from '@ar/components/TreeView/TreeNodeContent'
import { TreeViewContext } from '@ar/components/TreeView/TreeViewContext'
import type { RepositoryTreeNodeProps } from '@ar/frameworks/RepositoryStep/Repository'
import RepositoryActionsWidget from '@ar/frameworks/RepositoryStep/RepositoryActionsWidget'
import { PageType, type RepositoryConfigType, type RepositoryPackageType } from '@ar/common/types'
import ArtifactListTreeView from '@ar/pages/artifact-list/components/ArtifactListTreeView/ArtifactListTreeView'

import { RepositoryDetailsTab } from '../../constants'

interface IRepositoryTreeNode extends RepositoryTreeNodeProps {
  icon: IconName
  iconSize?: number
  level?: number
}
export default function RepositoryTreeNode(props: IRepositoryTreeNode) {
  const { data, icon, iconSize = 24, level = 0, isLastChild } = props
  const { setActivePath, activePath, compact } = useContext(TreeViewContext)
  const { useQueryParams } = useParentHooks()
  const queryParams = useQueryParams<Record<string, string>>()
  const routes = useRoutes()
  const history = useHistory()
  const path = data.identifier
  return (
    <TreeNode
      key={path}
      id={path}
      level={level}
      compact={compact}
      parentNodeLevels={[]}
      nodeType={NodeTypeEnum.Folder}
      isOpen={activePath.includes(path)}
      isActive={activePath === path}
      isLastChild={isLastChild}
      onClick={() => {
        setActivePath(path)
        history.push(
          routes.toARRepositoryDetailsTab(
            {
              repositoryIdentifier: data.identifier,
              tab: RepositoryDetailsTab.CONFIGURATION
            },
            { queryParams: omit(queryParams, 'digest') }
          )
        )
      }}
      heading={
        <TreeNodeContent
          icon={icon}
          iconSize={iconSize}
          label={data.identifier}
          type={data.type as RepositoryConfigType}
          artifacts={defaultTo(data.artifactsCount, 0)}
          downloads={defaultTo(data.downloadsCount, 0)}
          size={data.registrySize}
          compact={compact}
        />
      }
      actionElement={
        <RepositoryActionsWidget
          packageType={data.packageType as RepositoryPackageType}
          readonly={false}
          data={data}
          type={data.type as RepositoryConfigType}
          pageType={PageType.Table}
        />
      }>
      <ArtifactListTreeView registryIdentifier={data.identifier} parentNodeLevels={[{ data, isLastChild }]} />
    </TreeNode>
  )
}
