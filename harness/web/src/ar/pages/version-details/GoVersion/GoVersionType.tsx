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
import { Layout } from '@harnessio/uicore'
import type { ArtifactVersionSummary } from '@harnessio/react-har-service-client'

import { String } from '@ar/frameworks/strings'
import { PageType, RepositoryPackageType } from '@ar/common/types'
import { VersionListColumnEnum } from '@ar/pages/version-list/components/VersionListTable/types'
import ArtifactActions from '@ar/pages/artifact-details/components/ArtifactActions/ArtifactActions'
import ArtifactTreeNode from '@ar/pages/artifact-details/components/ArtifactTreeNode/ArtifactTreeNode'
import ArtifactTreeNodeDetailsContent from '@ar/pages/artifact-details/components/ArtifactTreeNode/ArtifactTreeNodeDetailsContent'
import VersionListTable, {
  type CommonVersionListTableProps
} from '@ar/pages/version-list/components/VersionListTable/VersionListTable'
import {
  type ArtifactActionProps,
  type ArtifactRowSubComponentProps,
  type ArtifactTreeNodeViewProps,
  type VersionActionProps,
  type VersionDetailsHeaderProps,
  type VersionDetailsTabProps,
  type VersionListTableProps,
  VersionStep,
  VersionTreeNodeViewProps
} from '@ar/frameworks/Version/Version'

import VersionFilesProvider from '../context/VersionFilesProvider'
import { VersionAction } from '../components/VersionActions/types'
import VersionActions from '../components/VersionActions/VersionActions'
import VersionOverviewProvider from '../context/VersionOverviewProvider'
import GoVersionOverviewPage from './pages/overview/GoVersionOverviewPage'
import { VersionDetailsTab } from '../components/VersionDetailsTabs/constants'
import ArtifactFilesContent from '../components/ArtifactFileListTable/ArtifactFilesContent'
import GoVersionArtifactDetailsPage from './pages/artifact-dertails/GoVersionArtifactDetailsPage'
import VersionDetailsHeaderContent from '../components/VersionDetailsHeaderContent/VersionDetailsHeaderContent'
import VersionTreeNode from '../components/VersionTreeNode/VersionTreeNode'
import VersionDetailsTabs from '../components/VersionDetailsTabs/VersionDetailsTabs'

export class GoVersionType extends VersionStep<ArtifactVersionSummary> {
  protected packageType = RepositoryPackageType.GO
  protected hasArtifactRowSubComponent = true
  protected allowedVersionDetailsTabs: VersionDetailsTab[] = [
    VersionDetailsTab.OVERVIEW,
    VersionDetailsTab.ARTIFACT_DETAILS,
    VersionDetailsTab.CODE
  ]

  versionListTableColumnConfig: CommonVersionListTableProps['columnConfigs'] = {
    [VersionListColumnEnum.Name]: { width: '150%' },
    [VersionListColumnEnum.Size]: { width: '100%' },
    [VersionListColumnEnum.FileCount]: { width: '100%' },
    [VersionListColumnEnum.DownloadCount]: { width: '100%' },
    [VersionListColumnEnum.PullCommand]: { width: '100%' },
    [VersionListColumnEnum.LastModified]: { width: '100%' },
    [VersionListColumnEnum.Actions]: { width: '30%' }
  }

  protected allowedActionsOnVersion = [
    VersionAction.Delete,
    VersionAction.SetupClient,
    VersionAction.ViewVersionDetails,
    VersionAction.Quarantine
  ]

  protected allowedActionsOnVersionDetailsPage = [VersionAction.Delete, VersionAction.Quarantine]

  renderVersionListTable(props: VersionListTableProps): JSX.Element {
    return <VersionListTable {...props} columnConfigs={this.versionListTableColumnConfig} />
  }

  renderVersionDetailsHeader(props: VersionDetailsHeaderProps<ArtifactVersionSummary>): JSX.Element {
    return <VersionDetailsHeaderContent {...props} />
  }

  renderVersionDetailsTab(props: VersionDetailsTabProps): JSX.Element {
    switch (props.tab) {
      case VersionDetailsTab.OVERVIEW:
        return (
          <VersionOverviewProvider>
            <GoVersionOverviewPage />
          </VersionOverviewProvider>
        )
      case VersionDetailsTab.ARTIFACT_DETAILS:
        return (
          <VersionOverviewProvider>
            <GoVersionArtifactDetailsPage />
          </VersionOverviewProvider>
        )
      case VersionDetailsTab.OSS:
        return (
          <VersionOverviewProvider>
            <Layout.Vertical spacing="xlarge">
              <GoVersionOverviewPage />
              <GoVersionArtifactDetailsPage />
            </Layout.Vertical>
          </VersionOverviewProvider>
        )
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
      <VersionFilesProvider
        repositoryIdentifier={props.data.registryIdentifier}
        artifactIdentifier={props.data.name}
        versionIdentifier={props.data.version}
        shouldUseLocalParams>
        <ArtifactFilesContent minimal />
      </VersionFilesProvider>
    )
  }

  renderArtifactTreeNodeView(props: ArtifactTreeNodeViewProps): JSX.Element {
    return <ArtifactTreeNode {...props} icon="store-artifact-bundle" />
  }

  renderArtifactTreeNodeDetails(): JSX.Element {
    return <ArtifactTreeNodeDetailsContent />
  }

  renderVersionTreeNodeView(props: VersionTreeNodeViewProps): JSX.Element {
    return <VersionTreeNode {...props} icon="container" />
  }

  renderVersionTreeNodeDetails(): JSX.Element {
    return <VersionDetailsTabs />
  }
}
