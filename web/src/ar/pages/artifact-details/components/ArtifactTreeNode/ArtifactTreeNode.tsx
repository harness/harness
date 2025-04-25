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
import { omit } from 'lodash-es'
import { useHistory } from 'react-router-dom'
import type { IconName } from '@harnessio/icons'
import type { RegistryArtifactMetadata, RegistryMetadata } from '@harnessio/react-har-service-client'

import { useParentHooks, useRoutes } from '@ar/hooks'
import TreeNode, { NodeTypeEnum } from '@ar/components/TreeView/TreeNode'
import TreeNodeContent from '@ar/components/TreeView/TreeNodeContent'
import { PageType, type RepositoryPackageType } from '@ar/common/types'
import { TreeViewContext } from '@ar/components/TreeView/TreeViewContext'
import type { ArtifactTreeNodeViewProps } from '@ar/frameworks/Version/Version'
import ArtifactActionsWidget from '@ar/frameworks/Version/ArtifactActionsWidget'
import VersionListTreeView from '@ar/pages/version-list/components/VersionListTreeView/VersionListTreeView'

interface IArtifactTreeNode extends ArtifactTreeNodeViewProps {
  icon: IconName
  iconSize?: number
  level?: number
}

export default function ArtifactTreeNode(props: IArtifactTreeNode) {
  const { data, icon, iconSize = 24, level = 1, isLastChild, parentNodeLevels } = props
  const { setActivePath, activePath, compact } = useContext(TreeViewContext)
  const { useQueryParams } = useParentHooks()
  const queryParams = useQueryParams<Record<string, string>>()
  const routes = useRoutes()
  const history = useHistory()
  const path = `${data.registryIdentifier}/${data.name}`
  return (
    <TreeNode<RegistryMetadata | RegistryArtifactMetadata>
      key={path}
      id={path}
      level={level}
      nodeType={NodeTypeEnum.Folder}
      compact={compact}
      isLastChild={isLastChild}
      parentNodeLevels={parentNodeLevels}
      isOpen={activePath.includes(path)}
      isActive={activePath === path}
      onClick={() => {
        setActivePath(path)
        history.push(
          routes.toARArtifactDetails(
            {
              repositoryIdentifier: data.registryIdentifier,
              artifactIdentifier: data.name
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
          downloads={data.downloadsCount}
          compact={compact}
        />
      }
      actionElement={
        <ArtifactActionsWidget
          packageType={data.packageType as RepositoryPackageType}
          data={data}
          repoKey={data.registryIdentifier}
          artifactKey={data.name}
          pageType={PageType.Table}
        />
      }>
      <VersionListTreeView
        parentNodeLevels={[...parentNodeLevels, { data, isLastChild }]}
        registryIdentifier={data.registryIdentifier}
        artifactIdentifier={data.name}
      />
    </TreeNode>
  )
}
