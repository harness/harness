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
import classNames from 'classnames'
import { Page } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import type { ArtifactDetailsPathParams } from '@ar/routes/types'
import { getIdentifierStringForBreadcrumb } from '@ar/common/utils'
import { useDecodedParams, useParentComponents, useRoutes } from '@ar/hooks'

import ArtifactDetailsHeaderContent from './ArtifactDetailsHeaderContent'

import css from './ArtifactDetailsHeader.module.scss'

function ArtifactDetailsHeader(): JSX.Element {
  const { repositoryIdentifier } = useDecodedParams<ArtifactDetailsPathParams>()
  const { NGBreadcrumbs } = useParentComponents()
  const { getString } = useStrings()
  const routes = useRoutes()

  return (
    <Page.Header
      title={<ArtifactDetailsHeaderContent />}
      className={classNames(css.header)}
      size="xlarge"
      breadcrumbs={
        <NGBreadcrumbs
          links={[
            {
              url: routes.toARRepositoryDetails({
                repositoryIdentifier
              }),
              label: getIdentifierStringForBreadcrumb(getString('breadcrumbs.repositories'), repositoryIdentifier)
            },
            {
              url: routes.toARArtifacts(),
              label: getString('breadcrumbs.artifacts')
            }
          ]}
        />
      }
    />
  )
}

export default ArtifactDetailsHeader
