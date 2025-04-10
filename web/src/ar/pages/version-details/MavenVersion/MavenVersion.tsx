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
import { VersionListColumnEnum } from '@ar/pages/version-list/components/VersionListTable/types'
import VersionListTable, {
  type CommonVersionListTableProps
} from '@ar/pages/version-list/components/VersionListTable/VersionListTable'
import {
  type ArtifactActionProps,
  ArtifactRowSubComponentProps,
  type VersionActionProps,
  type VersionDetailsHeaderProps,
  type VersionDetailsTabProps,
  type VersionListTableProps,
  VersionStep
} from '@ar/frameworks/Version/Version'

import OSSContentPage from './pages/oss-details/OSSContentPage'
import VersionFilesProvider from '../context/VersionFilesProvider'
import { VersionDetailsTab } from '../components/VersionDetailsTabs/constants'
import MavenArtifactOverviewPage from './pages/overview/MavenArtifactOverviewPage'
import MavenArtifactDetailsPage from './pages/artifact-details/MavenArtifactDetailsPage'
import ArtifactFilesContent from '../components/ArtifactFileListTable/ArtifactFilesContent'
import VersionDetailsHeaderContent from '../components/VersionDetailsHeaderContent/VersionDetailsHeaderContent'
import VersionActions from '../components/VersionActions/VersionActions'
import { VersionAction } from '../components/VersionActions/types'

export class MavenVersionType extends VersionStep<ArtifactVersionSummary> {
  protected packageType = RepositoryPackageType.MAVEN
  protected hasArtifactRowSubComponent = true
  protected allowedVersionDetailsTabs: VersionDetailsTab[] = [
    VersionDetailsTab.OVERVIEW,
    VersionDetailsTab.ARTIFACT_DETAILS,
    VersionDetailsTab.CODE
  ]

  versionListTableColumnConfig: CommonVersionListTableProps['columnConfigs'] = {
    [VersionListColumnEnum.Name]: { width: '100%' },
    [VersionListColumnEnum.Size]: { width: '100%' },
    [VersionListColumnEnum.FileCount]: { width: '100%' },
    [VersionListColumnEnum.LastModified]: { width: '100%' }
    // [VersionListColumnEnum.Actions]: { width: '10%' } // TODO: will add this once BE support actions
  }

  protected allowedActionsOnVersion = [VersionAction.SetupClient, VersionAction.ViewVersionDetails]
  protected allowedActionsOnVersionDetailsPage = []

  renderVersionListTable(props: VersionListTableProps): JSX.Element {
    return <VersionListTable {...props} columnConfigs={this.versionListTableColumnConfig} />
  }

  renderVersionDetailsHeader(props: VersionDetailsHeaderProps<ArtifactVersionSummary>): JSX.Element {
    return <VersionDetailsHeaderContent {...props} />
  }

  renderVersionDetailsTab(props: VersionDetailsTabProps): JSX.Element {
    switch (props.tab) {
      case VersionDetailsTab.OVERVIEW:
        return <MavenArtifactOverviewPage />
      case VersionDetailsTab.ARTIFACT_DETAILS:
        return <MavenArtifactDetailsPage />
      case VersionDetailsTab.OSS:
        return <OSSContentPage />
      default:
        return <String stringID="tabNotFound" />
    }
  }

  renderArtifactActions(_props: ArtifactActionProps): JSX.Element {
    return <></>
  }

  renderVersionActions(props: VersionActionProps): JSX.Element {
    const allowedActions =
      props.pageType === PageType.Table ? this.allowedActionsOnVersion : this.allowedActionsOnVersionDetailsPage
    return <VersionActions {...props} allowedActions={allowedActions} />
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
}
