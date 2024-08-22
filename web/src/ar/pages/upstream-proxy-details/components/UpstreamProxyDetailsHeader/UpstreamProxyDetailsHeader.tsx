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
import { useParentComponents, useRoutes } from '@ar/hooks'
import type { Repository } from '@ar/pages/repository-details/types'
import UpstreamProxyDetailsHeaderContent from './UpstreamProxyDetailsHeaderContent'
import css from './UpstreamProxyDetailsHeader.module.scss'

interface UpstreamProxyDetailsHeaderProps {
  data: Repository
}
export default function UpstreamProxyDetailsHeader(props: UpstreamProxyDetailsHeaderProps): JSX.Element {
  const { data } = props
  const { getString } = useStrings()
  const { NGBreadcrumbs } = useParentComponents()
  const routes = useRoutes()

  return (
    <Page.Header
      title={<UpstreamProxyDetailsHeaderContent data={data} iconSize={40} />}
      className={classNames(css.header)}
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
