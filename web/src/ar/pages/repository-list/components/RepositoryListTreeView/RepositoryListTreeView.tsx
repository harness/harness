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

import React, { useState } from 'react'
import { compact as lodashCompact } from 'lodash-es'
import { Switch } from 'react-router-dom'
import { Container, Layout } from '@harnessio/uicore'

import { useParentHooks, useRoutes } from '@ar/hooks'
import { PageType } from '@ar/common/types'
import RouteProvider from '@ar/components/RouteProvider/RouteProvider'
import RepositoryProvider from '@ar/pages/repository-details/context/RepositoryProvider'
import {
  artifactDetailsPathProps,
  repositoryDetailsPathProps,
  versionDetailsPathParams
} from '@ar/routes/RouteDestinations'
import { TreeViewContext } from '@ar/components/TreeView/TreeViewContext'
import VersionProvider from '@ar/pages/version-details/context/VersionProvider'
import ArtifactProvider from '@ar/pages/artifact-details/context/ArtifactProvider'
import type { DockerVersionDetailsQueryParams } from '@ar/pages/version-details/DockerVersion/types'
import VersionTreeNodeDetails from '@ar/pages/version-details/components/VersionTreeNode/VersionTreeNodeDetails'

import TreeView from '@ar/components/TreeView/TreeView'
import type { INodeConfig, ITreeNode } from '@ar/components/TreeView/types'
import VersionActionsWidget from '@ar/frameworks/Version/VersionActionsWidget'
import ArtifactActionsWidget from '@ar/frameworks/Version/ArtifactActionsWidget'
import RepositoryActionsWidget from '@ar/frameworks/RepositoryStep/RepositoryActionsWidget'
import ArtifactTreeNodeDetails from '@ar/pages/artifact-details/components/ArtifactTreeNode/ArtifactTreeNodeDetails'
import RepositoryTreeNodeDetails from '@ar/pages/repository-details/components/RepositoryTreeNode/RepositoryTreeNodeDetails'

import TreeNodeContent from '@ar/components/TreeView/TreeNodeContent'
import VersionTreeNodeViewWidget from '@ar/frameworks/Version/VersionTreeNodeViewWidget'
import ArtifactTreeNodeViewWidget from '@ar/frameworks/Version/ArtifactTreeNodeViewWidget'
import RepositoryTreeNodeViewWidget from '@ar/frameworks/RepositoryStep/RepositoryTreeNodeViewWidget'

import type { IGlobalFilters } from './types'
import { useRepositoryTreeViewUtils } from './utils'
import { TreeViewSortingOptions } from '../../constants'
import { useArtifactRepositoriesQueryParamOptions, type ArtifactRepositoryListPageQueryParams } from '../../utils'

import css from './RepositoryListTreeView.module.scss'

const ROOT_NODE_ID = 'root'
export default function RepositoryListTreeView() {
  const routeDefinitions = useRoutes(true)
  const { useQueryParams } = useParentHooks()
  const { digest } = useQueryParams<DockerVersionDetailsQueryParams>()
  const [activePath, setActivePath] = useState('')

  const queryParamOptions = useArtifactRepositoriesQueryParamOptions()
  const { registrySearchTerm, compact, repositoryTypes, configType, treeSort } =
    useQueryParams<ArtifactRepositoryListPageQueryParams>(queryParamOptions)

  const handleUpdateActivePath = (values: Record<string, string>) => {
    const initialActivePath = lodashCompact([
      ROOT_NODE_ID,
      values.repositoryIdentifier,
      values.artifactIdentifier ? encodeURIComponent(values.artifactIdentifier) : '',
      values.versionIdentifier,
      digest
    ]).join('/')
    setActivePath(initialActivePath)
  }

  const RepositoryTreeViewUtils = useRepositoryTreeViewUtils()

  const handleFetchTreeNodesByPath = async (node: ITreeNode, filters?: INodeConfig<IGlobalFilters>) => {
    const { metadata } = node
    const { repositoryIdentifier, artifactIdentifier, versionIdentifier } = metadata || {}
    if (versionIdentifier) {
      return RepositoryTreeViewUtils.fetchDockerDigestList(node, filters)
    } else if (artifactIdentifier) {
      return RepositoryTreeViewUtils.fetchArtifactVersionList(node, filters)
    } else if (repositoryIdentifier) {
      return RepositoryTreeViewUtils.fetchArtifactList(node, filters)
    } else {
      return RepositoryTreeViewUtils.fetchRegistryList(node, filters)
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
    <TreeViewContext.Provider value={{ activePath, setActivePath, compact }}>
      <Layout.Horizontal className={css.treeViewPageContainer}>
        <Container className={css.treeViewContainer}>
          <TreeView<IGlobalFilters>
            activePath={activePath}
            setActivePath={setActivePath}
            compact={compact}
            fetchData={handleFetchTreeNodesByPath}
            onClick={handleClickNode}
            renderNodeAction={renderNodeAction}
            renderNodeHeader={renderNodeLabel}
            globalSearchConfig={{
              className: css.searchInput,
              searchTerm: registrySearchTerm,
              sortOptions: TreeViewSortingOptions,
              sort: treeSort
            }}
            globalFilters={{
              repositoryTypes,
              configType
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
      </Layout.Horizontal>
    </TreeViewContext.Provider>
  )
}
