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
import type { ArtifactVersionSummary } from '@harnessio/react-har-service-client'

import { String } from '@ar/frameworks/strings'
import { NodeTypeEnum } from '@ar/components/TreeView/TreeNode'
import DigestListPage from '@ar/pages/digest-list/DigestListPage'
import { PageType, RepositoryPackageType } from '@ar/common/types'
import ArtifactActions from '@ar/pages/artifact-details/components/ArtifactActions/ArtifactActions'
import ArtifactTreeNode from '@ar/pages/artifact-details/components/ArtifactTreeNode/ArtifactTreeNode'
import DigestListTreeView from '@ar/pages/digest-list/components/DigestListTreeView/DigestListTreeView'
import DockerVersionListTable from '@ar/pages/version-list/DockerVersion/VersionListTable/DockerVersionListTable'
import ArtifactTreeNodeDetailsContent from '@ar/pages/artifact-details/components/ArtifactTreeNode/ArtifactTreeNodeDetailsContent'
import {
  type ArtifactActionProps,
  ArtifactRowSubComponentProps,
  type VersionActionProps,
  type ArtifactTreeNodeViewProps,
  type VersionDetailsHeaderProps,
  type VersionDetailsTabProps,
  type VersionListTableProps,
  VersionStep,
  type VersionTreeNodeViewProps
} from '@ar/frameworks/Version/Version'

import DockerVersionHeader from './DockerVersionHeader'
import { VersionAction } from '../components/VersionActions/types'
import DockerArtifactSSCAContent from './DockerArtifactSSCAContent'
import VersionActions from '../components/VersionActions/VersionActions'
import DockerVersionOverviewContent from './DockerVersionOverviewContent'
import DockerArtifactDetailsContent from './DockerArtifactDetailsContent'
import VersionTreeNode from '../components/VersionTreeNode/VersionTreeNode'
import { VersionDetailsTab } from '../components/VersionDetailsTabs/constants'
import DockerArtifactSecurityTestsContent from './DockerArtifactSecurityTestsContent'
import DockerVersionOSSContent from './DockerVersionOSSContent/DockerVersionOSSContent'
import DockerDeploymentsContent from './DockerDeploymentsContent/DockerDeploymentsContent'
import DockerVersionTreeNodeDetailsContent from './components/DockerVersionTreeNodeDetailsContent/DockerVersionTreeNodeDetailsContent'

export class DockerVersionType extends VersionStep<ArtifactVersionSummary> {
  protected packageType = RepositoryPackageType.DOCKER
  protected hasArtifactRowSubComponent = true
  protected allowedVersionDetailsTabs: VersionDetailsTab[] = [
    VersionDetailsTab.OVERVIEW,
    VersionDetailsTab.ARTIFACT_DETAILS,
    VersionDetailsTab.SUPPLY_CHAIN,
    VersionDetailsTab.SECURITY_TESTS,
    VersionDetailsTab.DEPLOYMENTS,
    VersionDetailsTab.CODE
  ]

  protected allowedActionsOnVersion = [
    VersionAction.Delete,
    VersionAction.SetupClient,
    VersionAction.DownloadCommand,
    VersionAction.ViewVersionDetails
  ]

  protected allowedActionsOnVersionDetailsPage = [VersionAction.Delete]

  renderVersionListTable(props: VersionListTableProps): JSX.Element {
    return <DockerVersionListTable {...props} />
  }

  renderVersionDetailsHeader(props: VersionDetailsHeaderProps<ArtifactVersionSummary>): JSX.Element {
    return <DockerVersionHeader data={props.data} />
  }

  renderVersionDetailsTab(props: VersionDetailsTabProps): JSX.Element {
    switch (props.tab) {
      case VersionDetailsTab.OVERVIEW:
        return <DockerVersionOverviewContent />
      case VersionDetailsTab.ARTIFACT_DETAILS:
        return <DockerArtifactDetailsContent />
      case VersionDetailsTab.OSS:
        return <DockerVersionOSSContent />
      case VersionDetailsTab.SECURITY_TESTS:
        return <DockerArtifactSecurityTestsContent />
      case VersionDetailsTab.SUPPLY_CHAIN:
        return <DockerArtifactSSCAContent />
      case VersionDetailsTab.DEPLOYMENTS:
        return <DockerDeploymentsContent />
      default:
        return <String stringID="tabNotFound" />
    }
  }

  renderArtifactActions(props: ArtifactActionProps): JSX.Element {
    return <ArtifactActions {...props} />
  }

  renderVersionActions(props: VersionActionProps): JSX.Element {
    switch (props.pageType) {
      case PageType.Details:
        return <VersionActions {...props} allowedActions={this.allowedActionsOnVersionDetailsPage} />
      case PageType.Table:
      case PageType.GlobalList:
      default:
        return <VersionActions {...props} allowedActions={this.allowedActionsOnVersion} />
    }
  }

  renderArtifactRowSubComponent(props: ArtifactRowSubComponentProps): JSX.Element {
    return (
      <DigestListPage repoKey={props.data.registryIdentifier} artifact={props.data.name} version={props.data.version} />
    )
  }

  renderArtifactTreeNodeView(props: ArtifactTreeNodeViewProps): JSX.Element {
    return <ArtifactTreeNode {...props} icon="store-artifact-bundle" />
  }

  renderArtifactTreeNodeDetails(): JSX.Element {
    return <ArtifactTreeNodeDetailsContent />
  }

  renderVersionTreeNodeView(props: VersionTreeNodeViewProps): JSX.Element {
    const { data, parentNodeLevels, isLastChild } = props
    return (
      <VersionTreeNode {...props} icon="container" nodeType={NodeTypeEnum.Folder}>
        <DigestListTreeView
          registryIdentifier={props.data.registryIdentifier}
          artifactIdentifier={props.artifactIdentifier}
          versionIdentifier={props.data.name}
          parentNodeLevels={[...parentNodeLevels, { data, isLastChild }]}
        />
      </VersionTreeNode>
    )
  }

  renderVersionTreeNodeDetails(): JSX.Element {
    return <DockerVersionTreeNodeDetailsContent />
  }
}
