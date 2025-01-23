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

import { RepositoryPackageType } from '@ar/common/types'
import { type VersionListTableProps, VersionStep } from '@ar/frameworks/Version/Version'

import { MavenVersionListPage } from './pages/list/MavenVersionListPage'
import { VersionDetailsTab } from '../components/VersionDetailsTabs/constants'

export class GenericVersionType extends VersionStep<ArtifactVersionSummary> {
  protected packageType = RepositoryPackageType.MAVEN
  protected allowedVersionDetailsTabs: VersionDetailsTab[] = [
    VersionDetailsTab.OVERVIEW,
    VersionDetailsTab.ARTIFACT_DETAILS,
    VersionDetailsTab.CODE
  ]

  renderVersionListTable(props: VersionListTableProps): JSX.Element {
    return <MavenVersionListPage {...props} />
  }

  renderVersionDetailsHeader(): JSX.Element {
    return <></>
  }

  renderVersionDetailsTab(): JSX.Element {
    return <></>
  }
}
