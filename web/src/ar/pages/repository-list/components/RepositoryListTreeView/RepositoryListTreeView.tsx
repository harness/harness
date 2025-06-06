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

import React, { useEffect, useState } from 'react'
import { compact as lodashCompact } from 'lodash-es'
import { Switch } from 'react-router-dom'
import { Container } from '@harnessio/uicore'

import { useAppStore, useParentHooks, useRoutes } from '@ar/hooks'
import { PageType } from '@ar/common/types'
import RouteProvider from '@ar/components/RouteProvider/RouteProvider'
import RepositoryProvider from '@ar/pages/repository-details/context/RepositoryProvider'
import {
  artifactDetailsPathProps,
  repositoryDetailsPathProps,
  versionDetailsPathParams
} from '@ar/routes/RouteDestinations'
import VersionProvider from '@ar/pages/version-details/context/VersionProvider'
import ArtifactProvider from '@ar/pages/artifact-details/context/ArtifactProvider'
import type { DockerVersionDetailsQueryParams } from '@ar/pages/version-details/DockerVersion/types'
import VersionTreeNodeDetails from '@ar/pages/version-details/components/VersionTreeNode/VersionTreeNodeDetails'

import TreeView from '@ar/components/TreeView/TreeView'
import type { INode, ITreeNode } from '@ar/components/TreeView/types'
import VersionActionsWidget from '@ar/frameworks/Version/VersionActionsWidget'
import ArtifactActionsWidget from '@ar/frameworks/Version/ArtifactActionsWidget'
import RepositoryActionsWidget from '@ar/frameworks/RepositoryStep/RepositoryActionsWidget'
import ArtifactTreeNodeDetails from '@ar/pages/artifact-details/components/ArtifactTreeNode/ArtifactTreeNodeDetails'
import RepositoryTreeNodeDetails from '@ar/pages/repository-details/components/RepositoryTreeNode/RepositoryTreeNodeDetails'

import TreeNodeContent from '@ar/components/TreeView/TreeNodeContent'
import VersionTreeNodeViewWidget from '@ar/frameworks/Version/VersionTreeNodeViewWidget'
import ArtifactTreeNodeViewWidget from '@ar/frameworks/Version/ArtifactTreeNodeViewWidget'
import RepositoryTreeNodeViewWidget from '@ar/frameworks/RepositoryStep/RepositoryTreeNodeViewWidget'

import { Split } from 'components/Split/Split'

import { useRepositoryTreeViewUtils } from './utils'
import { TreeViewSortingOptions } from '../../constants'
import { type TreeViewRepositoryQueryParams, useTreeViewRepositoriesQueryParamOptions } from '../../utils'

import css from './RepositoryListTreeView.module.scss'

const ROOT_NODE_ID = 'root'
export default function RepositoryListTreeView() {
  const routeDefinitions = useRoutes(true)
  const { useQueryParams, useUpdateQueryParams } = useParentHooks()
  const { digest } = useQueryParams<DockerVersionDetailsQueryParams>()
  const { updateQueryParams } = useUpdateQueryParams<Partial<TreeViewRepositoryQueryParams>>()
  const [activePath, setActivePath] = useState('')
  const [initialised, setInitialised] = useState(false)
  const { scope } = useAppStore()
  const { space } = scope

  const [rootNodes, setRootNodes] = useState<INode[]>([])
  const [loadingRootNodes, setLoadingRootNodes] = useState<boolean>(false)

  const queryParamOptions = useTreeViewRepositoriesQueryParamOptions()
  const queryParams = useQueryParams<TreeViewRepositoryQueryParams>(queryParamOptions)
  const { searchTerm, compact, packageTypes, configType, sort } = queryParams
  const RepositoryTreeViewUtils = useRepositoryTreeViewUtils(queryParams)

  const fetchRootNodes = async () => {
    const rooteNode = { id: ROOT_NODE_ID } as ITreeNode
    try {
      setLoadingRootNodes(true)
      const registryList = await RepositoryTreeViewUtils.fetchRegistryList(rooteNode)
      setRootNodes(registryList)
    } catch (e) {
      const errorNodeConfig = RepositoryTreeViewUtils.getErrorNodeNodeConfig(rooteNode, e as string)
      setRootNodes([errorNodeConfig])
    } finally {
      setLoadingRootNodes(false)
    }
  }

  useEffect(() => {
    fetchRootNodes()
  }, [searchTerm, packageTypes, configType, sort, space])

  const handleUpdateActivePath = (values: Record<string, string>) => {
    const initialActivePath = lodashCompact([
      ROOT_NODE_ID,
      values.repositoryIdentifier,
      values.artifactIdentifier ? encodeURIComponent(values.artifactIdentifier) : '',
      values.versionIdentifier,
      digest
    ]).join('/')
    setActivePath(initialActivePath)
    setInitialised(true)
  }

  const handleFetchTreeNodesByPath = async (node: ITreeNode): Promise<Array<INode>> => {
    const { metadata } = node
    const { repositoryIdentifier, artifactIdentifier, versionIdentifier } = metadata || {}
    if (versionIdentifier) {
      return RepositoryTreeViewUtils.fetchDockerDigestList(node, node.metadata)
    } else if (artifactIdentifier) {
      return RepositoryTreeViewUtils.fetchArtifactVersionList(node, node.metadata)
    } else if (repositoryIdentifier) {
      return RepositoryTreeViewUtils.fetchArtifactList(node, node.metadata)
    } else {
      return RepositoryTreeViewUtils.fetchRegistryList(node, node.metadata)
    }
  }

  const handleClickNode = (node: ITreeNode) => {
    const { id: pathId, metadata } = node
    const { repositoryIdentifier, artifactIdentifier, versionIdentifier, digestIdentifier } = metadata || {}
    setActivePath(pathId)
    if (digestIdentifier) {
      return RepositoryTreeViewUtils.handleNavigateToDigestDetails(node)
    } else if (versionIdentifier) {
      return RepositoryTreeViewUtils.handleNavigateToVersionDetails(node)
    } else if (artifactIdentifier) {
      return RepositoryTreeViewUtils.handleNavigateToArtifactDetials(node)
    } else if (repositoryIdentifier) {
      return RepositoryTreeViewUtils.handleNavigateToRepositoryDetials(node)
    }
  }

  const renderNodeAction = (node: ITreeNode): JSX.Element => {
    const { metadata } = node
    const { repositoryIdentifier, artifactIdentifier, versionIdentifier, digestIdentifier } = metadata || {}
    if (digestIdentifier) {
      return <></>
    } else if (versionIdentifier) {
      return (
        <VersionActionsWidget
          packageType={metadata.packageType}
          data={node.metadata}
          repoKey={repositoryIdentifier}
          artifactKey={artifactIdentifier}
          versionKey={versionIdentifier}
          pageType={PageType.GlobalList}
        />
      )
    } else if (artifactIdentifier) {
      return (
        <ArtifactActionsWidget
          packageType={metadata.packageType}
          data={node.metadata}
          repoKey={repositoryIdentifier}
          artifactKey={artifactIdentifier}
          pageType={PageType.Table}
        />
      )
    } else if (repositoryIdentifier) {
      return (
        <RepositoryActionsWidget
          packageType={metadata.packageType}
          data={node.metadata}
          pageType={PageType.Table}
          type={metadata.type}
          readonly={false}
        />
      )
    }
    return <></>
  }

  const renderNodeLabel = (node: ITreeNode): JSX.Element => {
    const { metadata } = node
    const { repositoryIdentifier, artifactIdentifier, versionIdentifier, digestIdentifier } = metadata || {}
    if (digestIdentifier) {
      return <TreeNodeContent label={node.label} compact={compact} />
    } else if (versionIdentifier) {
      return <VersionTreeNodeViewWidget data={node.metadata} packageType={node.metadata.packageType} />
    } else if (artifactIdentifier) {
      return <ArtifactTreeNodeViewWidget data={node.metadata} packageType={node.metadata.packageType} />
    } else if (repositoryIdentifier) {
      return <RepositoryTreeNodeViewWidget data={node.metadata} packageType={node.metadata.packageType} />
    }
    return <></>
  }

  return (
    <Container className={css.treeViewPageContainer}>
      <Split className={css.splitPane} split="vertical" size={400} minSize={200} maxSize={500}>
        <Container className={css.treeViewContainer}>
          <TreeView
            activePath={activePath}
            setActivePath={setActivePath}
            rootNodes={rootNodes}
            loadingRootNodes={loadingRootNodes}
            compact={compact}
            fetchData={handleFetchTreeNodesByPath}
            onClick={handleClickNode}
            renderNodeAction={renderNodeAction}
            renderNodeHeader={renderNodeLabel}
            initialised={initialised}
            globalSearchConfig={{
              className: css.searchInput,
              searchTerm: searchTerm,
              sortOptions: TreeViewSortingOptions,
              sort: sort,
              onChange: (query: string, sortVal?: string) => {
                updateQueryParams({
                  searchTerm: query,
                  sort: sortVal
                })
              }
            }}
            rootPath={ROOT_NODE_ID}
          />
        </Container>
        <Container className={css.treeViewPageContentContainer}>
          <Switch>
            <RouteProvider exact onLoad={handleUpdateActivePath} path={[routeDefinitions.toARRepositories()]}>
              {/* TODO: Implement a default page for this path */}
              <></>
            </RouteProvider>
            <RouteProvider
              exact
              onLoad={handleUpdateActivePath}
              path={[routeDefinitions.toARArtifactDetails({ ...artifactDetailsPathProps })]}>
              <ArtifactProvider>
                <ArtifactTreeNodeDetails />
              </ArtifactProvider>
            </RouteProvider>
            <RouteProvider
              onLoad={handleUpdateActivePath}
              path={[routeDefinitions.toARVersionDetails({ ...versionDetailsPathParams })]}>
              <VersionProvider>
                <VersionTreeNodeDetails />
              </VersionProvider>
            </RouteProvider>
            <RouteProvider
              onLoad={handleUpdateActivePath}
              path={[routeDefinitions.toARRepositoryDetails({ ...repositoryDetailsPathProps })]}>
              <RepositoryProvider>
                <RepositoryTreeNodeDetails />
              </RepositoryProvider>
            </RouteProvider>
          </Switch>
        </Container>
      </Split>
    </Container>
  )
}
