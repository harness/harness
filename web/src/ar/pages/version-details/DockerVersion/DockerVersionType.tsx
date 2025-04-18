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
import DigestListPage from '@ar/pages/digest-list/DigestListPage'
import ArtifactActions from '@ar/pages/artifact-details/components/ArtifactActions/ArtifactActions'
import DockerVersionListTable from '@ar/pages/version-list/DockerVersion/VersionListTable/DockerVersionListTable'
import {
  type ArtifactActionProps,
  ArtifactRowSubComponentProps,
  type VersionActionProps,
  type VersionDetailsHeaderProps,
  type VersionDetailsTabProps,
  type VersionListTableProps,
  VersionStep
} from '@ar/frameworks/Version/Version'

import DockerVersionHeader from './DockerVersionHeader'
import DockerArtifactSSCAContent from './DockerArtifactSSCAContent'
import DockerVersionOverviewContent from './DockerVersionOverviewContent'
import DockerArtifactDetailsContent from './DockerArtifactDetailsContent'
import { VersionDetailsTab } from '../components/VersionDetailsTabs/constants'
import DockerArtifactSecurityTestsContent from './DockerArtifactSecurityTestsContent'
import DockerVersionOSSContent from './DockerVersionOSSContent/DockerVersionOSSContent'
import DockerDeploymentsContent from './DockerDeploymentsContent/DockerDeploymentsContent'
import VersionActions from '../components/VersionActions/VersionActions'
import { VersionAction } from '../components/VersionActions/types'

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
    const allowedActions =
      props.pageType === PageType.Table ? this.allowedActionsOnVersion : this.allowedActionsOnVersionDetailsPage
    return <VersionActions {...props} allowedActions={allowedActions} />
  }

  renderArtifactRowSubComponent(props: ArtifactRowSubComponentProps): JSX.Element {
    return (
      <DigestListPage repoKey={props.data.registryIdentifier} artifact={props.data.name} version={props.data.version} />
    )
  }
}
