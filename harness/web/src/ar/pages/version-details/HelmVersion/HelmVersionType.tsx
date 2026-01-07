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
import { PageType, RepositoryPackageType } from '@ar/common/types'
import ArtifactActions from '@ar/pages/artifact-details/components/ArtifactActions/ArtifactActions'
import ArtifactTreeNode from '@ar/pages/artifact-details/components/ArtifactTreeNode/ArtifactTreeNode'
import ArtifactTreeNodeDetailsContent from '@ar/pages/artifact-details/components/ArtifactTreeNode/ArtifactTreeNodeDetailsContent'
import {
  type ArtifactActionProps,
  type VersionActionProps,
  type ArtifactTreeNodeViewProps,
  type VersionDetailsHeaderProps,
  type VersionDetailsTabProps,
  type VersionListTableProps,
  VersionStep,
  type VersionTreeNodeViewProps
} from '@ar/frameworks/Version/Version'

import HelmVersionOverviewContent from './HelmVersionOverviewContent'
import HelmArtifactDetailsContent from './HelmArtifactDetailsContent'
import VersionActions from '../components/VersionActions/VersionActions'
import VersionTreeNode from '../components/VersionTreeNode/VersionTreeNode'
import { VersionDetailsTab } from '../components/VersionDetailsTabs/constants'
import HelmVersionOSSContent from './HelmVersionOSSContent/HelmVersionOSSContent'
import VersionDetailsTabs from '../components/VersionDetailsTabs/VersionDetailsTabs'
import { VersionAction } from '../components/VersionActions/types'
import HelmVersionListTable from './components/HelmVersionListTable/HelmVersionListTable'
import HelmVersionDetailsHeaderContent from './components/HelmVersionDetailsHeaderContent/HelmVersionDetailsHeaderContent'

export class HelmVersionType extends VersionStep<ArtifactVersionSummary> {
  protected packageType = RepositoryPackageType.HELM
  protected hasArtifactRowSubComponent = false
  protected allowedVersionDetailsTabs: VersionDetailsTab[] = [
    VersionDetailsTab.OVERVIEW,
    VersionDetailsTab.ARTIFACT_DETAILS,
    VersionDetailsTab.CODE
  ]

  protected allowedActionsOnVersion = [
    VersionAction.Delete,
    VersionAction.SetupClient,
    VersionAction.DownloadCommand,
    VersionAction.ViewVersionDetails,
    VersionAction.Quarantine
  ]

  protected allowedActionsOnVersionDetailsPage = [VersionAction.Delete, VersionAction.Quarantine]

  renderVersionListTable(props: VersionListTableProps): JSX.Element {
    return <HelmVersionListTable {...props} />
  }

  renderVersionDetailsHeader(props: VersionDetailsHeaderProps<ArtifactVersionSummary>): JSX.Element {
    return <HelmVersionDetailsHeaderContent data={props.data} />
  }

  renderVersionDetailsTab(props: VersionDetailsTabProps): JSX.Element {
    switch (props.tab) {
      case VersionDetailsTab.OVERVIEW:
        return <HelmVersionOverviewContent />
      case VersionDetailsTab.ARTIFACT_DETAILS:
        return <HelmArtifactDetailsContent />
      case VersionDetailsTab.OSS:
        return <HelmVersionOSSContent />
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

  renderArtifactRowSubComponent(): JSX.Element {
    return <></>
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
