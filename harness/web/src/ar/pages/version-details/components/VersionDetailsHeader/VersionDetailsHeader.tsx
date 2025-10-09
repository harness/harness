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

import React, { useContext } from 'react'
import classNames from 'classnames'
import { Page } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import { useDecodedParams, useRoutes } from '@ar/hooks'
import type { RepositoryPackageType } from '@ar/common/types'
import type { VersionDetailsPathParams } from '@ar/routes/types'
import Breadcrumbs from '@ar/components/Breadcrumbs/Breadcrumbs'
import { getIdentifierStringForBreadcrumb } from '@ar/common/utils'
import { VersionProviderContext } from '@ar/pages/version-details/context/VersionProvider'

import { LocalArtifactType } from '@ar/pages/repository-details/constants'
import VersionDetailsHeaderWidget from '@ar/frameworks/Version/VersionDetailsHeaderWidget'
import css from './VersionDetailsHeader.module.scss'

interface VersionDetailsHeaderProps {
  className?: string
}

export function VersionDetailsHeader(props: VersionDetailsHeaderProps): JSX.Element | null {
  const { repositoryIdentifier, artifactIdentifier } = useDecodedParams<VersionDetailsPathParams>()
  const { data } = useContext(VersionProviderContext)
  const { getString } = useStrings()
  const routes = useRoutes()
  if (!data) return null
  return (
    <Page.Header
      title={<VersionDetailsHeaderWidget packageType={data.packageType as RepositoryPackageType} data={data} />}
      className={classNames(css.header, props.className)}
      size="large"
      breadcrumbs={
        <Breadcrumbs
          links={[
            {
              url: routes.toARRepositoryDetails({
                repositoryIdentifier
              }),
              label: getIdentifierStringForBreadcrumb(getString('breadcrumbs.repositories'), repositoryIdentifier)
            },
            {
              url: routes.toARArtifactDetails({
                repositoryIdentifier,
                artifactIdentifier,
                artifactType: (data?.artifactType ?? LocalArtifactType.ARTIFACTS) as LocalArtifactType
              }),
              label: getIdentifierStringForBreadcrumb(getString('breadcrumbs.artifacts'), artifactIdentifier)
            }
          ]}
        />
      }
    />
  )
}
