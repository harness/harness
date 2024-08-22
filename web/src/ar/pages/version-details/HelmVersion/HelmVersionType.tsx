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
import { RepositoryPackageType } from '@ar/common/types'
import VersionListTable from '@ar/pages/version-list/components/VersionListTable/VersionListTable'
import {
  VersionDetailsHeaderProps,
  VersionDetailsTabProps,
  VersionListTableProps,
  VersionStep
} from '@ar/frameworks/Version/Version'
import HelmVersionHeader from './HelmVersionHeader'
import { VersionDetailsTab } from '../components/VersionDetailsTabs/constants'
import HelmVersionOverviewContent from './HelmVersionOverviewContent'
import HelmArtifactDetailsContent from './HelmArtifactDetailsContent'
import HelmVersionOSSContent from './HelmVersionOSSContent/HelmVersionOSSContent'

export class HelmVersionType extends VersionStep<ArtifactVersionSummary> {
  protected packageType = RepositoryPackageType.HELM
  protected allowedVersionDetailsTabs: VersionDetailsTab[] = [
    VersionDetailsTab.OVERVIEW,
    VersionDetailsTab.ARTIFACT_DETAILS,
    VersionDetailsTab.DEPLOYMENTS,
    VersionDetailsTab.CODE
  ]

  renderVersionListTable(props: VersionListTableProps): JSX.Element {
    return <VersionListTable {...props} />
  }

  renderVersionDetailsHeader(props: VersionDetailsHeaderProps<ArtifactVersionSummary>): JSX.Element {
    return <HelmVersionHeader data={props.data} />
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
}
