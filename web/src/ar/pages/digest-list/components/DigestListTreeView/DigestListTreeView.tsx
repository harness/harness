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

import React, { useContext, useState } from 'react'
import { useHistory } from 'react-router-dom'
import {
  type ArtifactVersionMetadata,
  type DockerManifestDetails,
  type RegistryArtifactMetadata,
  type RegistryMetadata,
  useGetDockerArtifactManifestsQuery
} from '@harnessio/react-har-service-client'

import { useGetSpaceRef, useParentHooks, useRoutes } from '@ar/hooks'
import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import TreeNode, { NodeTypeEnum } from '@ar/components/TreeView/TreeNode'
import TreeBody from '@ar/components/TreeView/TreeBody'
import TreeNodeList from '@ar/components/TreeView/TreeNodeList'
import TreeNodeContent from '@ar/components/TreeView/TreeNodeContent'
import { type NodeSpec, TreeViewContext } from '@ar/components/TreeView/TreeViewContext'
import { VersionDetailsTab } from '@ar/pages/version-details/components/VersionDetailsTabs/constants'

import { getShortDigest } from '../../utils'

interface DigestListTreeViewProps {
  registryIdentifier: string
  artifactIdentifier: string
  versionIdentifier: string
  parentNodeLevels: Array<NodeSpec<RegistryMetadata | RegistryArtifactMetadata | ArtifactVersionMetadata>>
}

export default function DigestListTreeView(props: DigestListTreeViewProps) {
  const { registryIdentifier, artifactIdentifier, versionIdentifier, parentNodeLevels } = props
  const [page] = useState(0)
  const [searchTerm] = useState('')
  const routes = useRoutes()
  const history = useHistory()
  const { setActivePath, activePath, compact } = useContext(TreeViewContext)
  const { useQueryParams } = useParentHooks()
  const queryParams = useQueryParams<Record<string, string>>()

  const registryRef = useGetSpaceRef(registryIdentifier)
  const {
    data,
    refetch,
    isFetching: loading,
    error
  } = useGetDockerArtifactManifestsQuery({
    registry_ref: registryRef,
    artifact: encodeRef(artifactIdentifier),
    version: versionIdentifier,
    queryParams: {
      page,
      size: 100,
      search_term: searchTerm
    },
    stringifyQueryParamsOptions: {
      arrayFormat: 'repeat'
    }
  })
  const manifestList = data?.content.data.manifests || []
  return (
    <TreeNodeList>
      <TreeBody
        loading={loading}
        error={error?.message}
        retryOnError={refetch}
        isEmpty={!manifestList.length}
        emptyDataMessage="digestList.table.noDigestTitle">
        {manifestList.map((each, idx) => {
          const path = `${registryIdentifier}/${artifactIdentifier}/${versionIdentifier}/${each.digest}`
          const isLastChild = idx === manifestList.length - 1
          return (
            <TreeNode<
              | RegistryMetadata
              | RegistryArtifactMetadata
              | ArtifactVersionMetadata
              | ArtifactVersionMetadata
              | DockerManifestDetails
            >
              key={path}
              id={path}
              level={3}
              compact={compact}
              nodeType={NodeTypeEnum.File}
              isOpen={activePath.includes(path)}
              isActive={activePath === path}
              isLastChild={isLastChild}
              parentNodeLevels={parentNodeLevels}
              onClick={() => {
                setActivePath(path)
                history.push(
                  routes.toARVersionDetailsTab(
                    {
                      repositoryIdentifier: registryIdentifier,
                      artifactIdentifier,
                      versionIdentifier,
                      versionTab: VersionDetailsTab.OVERVIEW
                    },
                    {
                      queryParams: {
                        ...queryParams,
                        digest: each.digest
                      }
                    }
                  )
                )
              }}
              heading={
                <TreeNodeContent
                  icon="file"
                  iconSize={20}
                  size={each.size}
                  compact={compact}
                  label={getShortDigest(each.digest)}
                  downloads={each.downloadsCount}
                />
              }
            />
          )
        })}
      </TreeBody>
    </TreeNodeList>
  )
}
