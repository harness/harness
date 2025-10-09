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
  repositoryDetailsTabPathProps,
  versionDetailsPathParams,
  versionDetailsTabPathParams
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

import { TreeNodeEntityEnum } from './types'
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
      values.artifactType,
      values.artifactIdentifier ? encodeURIComponent(values.artifactIdentifier) : '',
      values.versionIdentifier,
      digest
    ]).join('/')
    setActivePath(initialActivePath)
    setInitialised(true)
  }

  const handleFetchTreeNodesByPath = async (node: ITreeNode): Promise<Array<INode>> => {
    const { metadata } = node
    const { entityType, artifactType } = metadata || {}
    switch (entityType) {
      case TreeNodeEntityEnum.REGISTRY: {
        if (artifactType) {
          return RepositoryTreeViewUtils.fetchArtifactList(node, node.metadata)
        }
        return RepositoryTreeViewUtils.fetchArtifactTypeList(node, node.metadata)
      }
      case TreeNodeEntityEnum.ARTIFACT_TYPE:
        return RepositoryTreeViewUtils.fetchArtifactList(node, node.metadata)
      case TreeNodeEntityEnum.ARTIFACT:
        return RepositoryTreeViewUtils.fetchArtifactVersionList(node, node.metadata)
      case TreeNodeEntityEnum.VERSION:
        return RepositoryTreeViewUtils.fetchDockerDigestList(node, node.metadata)
      default:
        return RepositoryTreeViewUtils.fetchRegistryList(node, node.metadata)
    }
  }

  const handleClickNode = (node: ITreeNode) => {
    const { id: pathId, metadata } = node
    const { entityType } = metadata || {}
    setActivePath(pathId)
    switch (entityType) {
      case TreeNodeEntityEnum.REGISTRY:
        return RepositoryTreeViewUtils.handleNavigateToRepositoryDetials(node)
      case TreeNodeEntityEnum.ARTIFACT:
        return RepositoryTreeViewUtils.handleNavigateToArtifactDetials(node)
      case TreeNodeEntityEnum.VERSION:
        return RepositoryTreeViewUtils.handleNavigateToVersionDetails(node)
      case TreeNodeEntityEnum.DIGEST:
        return RepositoryTreeViewUtils.handleNavigateToDigestDetails(node)
      case TreeNodeEntityEnum.ARTIFACT_TYPE:
      default:
        return
    }
  }

  const renderNodeAction = (node: ITreeNode): JSX.Element => {
    const { metadata } = node
    const { entityType, repositoryIdentifier, artifactIdentifier, versionIdentifier } = metadata || {}
    switch (entityType) {
      case TreeNodeEntityEnum.REGISTRY:
        return (
          <RepositoryActionsWidget
            packageType={metadata.packageType}
            data={node.metadata}
            pageType={PageType.Table}
            type={metadata.type}
            readonly={false}
          />
        )
      case TreeNodeEntityEnum.ARTIFACT:
        return (
          <ArtifactActionsWidget
            packageType={metadata.packageType}
            data={node.metadata}
            repoKey={repositoryIdentifier}
            artifactKey={artifactIdentifier}
            pageType={PageType.Table}
          />
        )
      case TreeNodeEntityEnum.VERSION:
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
      case TreeNodeEntityEnum.ARTIFACT_TYPE:
      case TreeNodeEntityEnum.DIGEST:
      default:
        return <></>
    }
  }

  const renderNodeLabel = (node: ITreeNode): JSX.Element => {
    const { metadata } = node
    const { entityType } = metadata || {}
    switch (entityType) {
      case TreeNodeEntityEnum.REGISTRY:
        return <RepositoryTreeNodeViewWidget data={node.metadata} packageType={node.metadata.packageType} />
      case TreeNodeEntityEnum.ARTIFACT_TYPE:
        return <TreeNodeContent icon="folder" label={node.label} compact={compact} />
      case TreeNodeEntityEnum.ARTIFACT:
        return <ArtifactTreeNodeViewWidget data={node.metadata} packageType={node.metadata.packageType} />
      case TreeNodeEntityEnum.VERSION:
        return <VersionTreeNodeViewWidget data={node.metadata} packageType={node.metadata.packageType} />
      case TreeNodeEntityEnum.DIGEST:
        return <TreeNodeContent label={node.label} compact={compact} />
      default:
        return <TreeNodeContent label={node.label} compact={compact} />
    }
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
              onLoad={handleUpdateActivePath}
              exact
              path={[
                routeDefinitions.toARRepositoryDetails({ ...repositoryDetailsPathProps }),
                routeDefinitions.toARRepositoryDetailsTab({ ...repositoryDetailsTabPathProps })
              ]}>
              <RepositoryProvider>
                <RepositoryTreeNodeDetails />
              </RepositoryProvider>
            </RouteProvider>
            <RouteProvider
              exact
              onLoad={handleUpdateActivePath}
              path={[
                routeDefinitions.toARVersionDetails({ ...versionDetailsPathParams }),
                routeDefinitions.toARVersionDetailsTab({ ...versionDetailsTabPathParams })
              ]}>
              <VersionProvider>
                <VersionTreeNodeDetails />
              </VersionProvider>
            </RouteProvider>
            <RouteProvider
              exact
              onLoad={handleUpdateActivePath}
              path={[routeDefinitions.toARArtifactDetails({ ...artifactDetailsPathProps })]}>
              <ArtifactProvider>
                <ArtifactTreeNodeDetails />
              </ArtifactProvider>
            </RouteProvider>
          </Switch>
        </Container>
      </Split>
    </Container>
  )
}
