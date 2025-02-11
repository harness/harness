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
import { VersionListColumnEnum } from '@ar/pages/version-list/components/VersionListTable/types'
import VersionListTable, {
  CommonVersionListTableProps
} from '@ar/pages/version-list/components/VersionListTable/VersionListTable'
import {
  VersionDetailsHeaderProps,
  VersionDetailsTabProps,
  VersionListTableProps,
  VersionStep
} from '@ar/frameworks/Version/Version'
import { VersionDetailsTab } from '../components/VersionDetailsTabs/constants'
import HelmVersionOverviewContent from './HelmVersionOverviewContent'
import HelmArtifactDetailsContent from './HelmArtifactDetailsContent'
import HelmVersionOSSContent from './HelmVersionOSSContent/HelmVersionOSSContent'
import VersionDetailsHeaderContent from '../components/VersionDetailsHeaderContent/VersionDetailsHeaderContent'

export class HelmVersionType extends VersionStep<ArtifactVersionSummary> {
  protected packageType = RepositoryPackageType.HELM
  protected allowedVersionDetailsTabs: VersionDetailsTab[] = [
    VersionDetailsTab.OVERVIEW,
    VersionDetailsTab.ARTIFACT_DETAILS,
    VersionDetailsTab.CODE
  ]
  versionListTableColumnConfig: CommonVersionListTableProps['columnConfigs'] = {
    [VersionListColumnEnum.Name]: { width: '30%' },
    [VersionListColumnEnum.Size]: { width: '8%' },
    [VersionListColumnEnum.DownloadCount]: { width: '10%' },
    [VersionListColumnEnum.LastModified]: { width: '12%' },
    [VersionListColumnEnum.PullCommand]: { width: '40%' }
  }

  renderVersionListTable(props: VersionListTableProps): JSX.Element {
    return <VersionListTable {...props} columnConfigs={this.versionListTableColumnConfig} />
  }

  renderVersionDetailsHeader(props: VersionDetailsHeaderProps<ArtifactVersionSummary>): JSX.Element {
    return <VersionDetailsHeaderContent data={props.data} />
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
