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

import React, { PropsWithChildren, useContext } from 'react'
import { omit } from 'lodash-es'
import { useHistory } from 'react-router-dom'
import type { IconName } from '@harnessio/icons'
import type {
  ArtifactVersionMetadata,
  RegistryArtifactMetadata,
  RegistryMetadata
} from '@harnessio/react-har-service-client'

import { useParentHooks, useRoutes } from '@ar/hooks'
import TreeNode, { NodeTypeEnum } from '@ar/components/TreeView/TreeNode'
import TreeNodeContent from '@ar/components/TreeView/TreeNodeContent'
import { PageType, type RepositoryPackageType } from '@ar/common/types'
import { TreeViewContext } from '@ar/components/TreeView/TreeViewContext'
import VersionActionsWidget from '@ar/frameworks/Version/VersionActionsWidget'
import type { VersionTreeNodeViewProps } from '@ar/frameworks/Version/Version'
import { VersionDetailsTab } from '../VersionDetailsTabs/constants'

interface IVersionTreeNode extends VersionTreeNodeViewProps {
  icon: IconName
  iconSize?: number
  level?: number
  nodeType?: NodeTypeEnum
}

export default function VersionTreeNode(props: PropsWithChildren<IVersionTreeNode>) {
  const {
    data,
    icon,
    iconSize = 24,
    level = 2,
    nodeType = NodeTypeEnum.File,
    artifactIdentifier,
    isLastChild,
    parentNodeLevels
  } = props
  const { setActivePath, activePath, compact } = useContext(TreeViewContext)
  const routes = useRoutes()
  const history = useHistory()
  const { useQueryParams } = useParentHooks()
  const queryParams = useQueryParams<Record<string, string>>()
  const path = `${data.registryIdentifier}/${artifactIdentifier}/${data.name}`
  return (
    <TreeNode<RegistryMetadata | RegistryArtifactMetadata | ArtifactVersionMetadata>
      key={path}
      id={path}
      level={level}
      nodeType={nodeType}
      compact={compact}
      isOpen={activePath.includes(path)}
      isActive={activePath === path}
      isLastChild={isLastChild}
      parentNodeLevels={parentNodeLevels}
      onClick={() => {
        setActivePath(path)
        history.push(
          routes.toARVersionDetailsTab(
            {
              repositoryIdentifier: data.registryIdentifier,
              artifactIdentifier: artifactIdentifier,
              versionIdentifier: data.name,
              versionTab: VersionDetailsTab.OVERVIEW
            },
            { queryParams: omit(queryParams, 'digest') }
          )
        )
      }}
      heading={
        <TreeNodeContent
          icon={icon}
          iconSize={iconSize}
          label={data.name}
          size={data.size}
          downloads={data.downloadsCount}
          compact={compact}
        />
      }
      actionElement={
        <VersionActionsWidget
          pageType={PageType.Table}
          data={data}
          repoKey={data.registryIdentifier}
          artifactKey={artifactIdentifier}
          versionKey={data.name}
          packageType={data.packageType as RepositoryPackageType}
        />
      }>
      {nodeType === NodeTypeEnum.Folder && props.children}
    </TreeNode>
  )
}
