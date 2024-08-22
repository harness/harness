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
import { Page } from '@harnessio/uicore'
import cx from 'classnames'

import { useParentComponents, useRoutes } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import type { Repository } from '@ar/pages/repository-details/types'
import RepositoryDetailsHeaderContent from './RepositoryDetailsHeaderContent'

import css from './RepositoryDetailsHeader.module.scss'

interface RepositoryDetailsHeaderProps {
  data: Repository
}

export default function RepositoryDetailsHeader(props: RepositoryDetailsHeaderProps): JSX.Element {
  const { data } = props
  const { NGBreadcrumbs } = useParentComponents()

  const { getString } = useStrings()
  const routes = useRoutes()

  return (
    <Page.Header
      title={<RepositoryDetailsHeaderContent data={data} iconSize={40} />}
      className={cx(css.header)}
      size="xlarge"
      breadcrumbs={
        <NGBreadcrumbs
          links={[
            {
              url: routes.toARRepositories(),
              label: getString('breadcrumbs.repositories')
            }
          ]}
        />
      }
    />
  )
}
