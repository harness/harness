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

import { useRoutes } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import Breadcrumbs from '@ar/components/Breadcrumbs/Breadcrumbs'

import ExemptionDetailsHeaderContent from './ExemptionDetailsHeaderContent'
import { ExemptionDetailsContext } from '../../context/ExemptionDetailsProvider'

import css from './ExemptionDetailsHeader.module.scss'

export default function ExemptionDetailsHeader(): JSX.Element {
  const { getString } = useStrings()
  const routes = useRoutes()
  const { data } = useContext(ExemptionDetailsContext)

  return (
    <Page.Header
      title={<ExemptionDetailsHeaderContent data={data} />}
      className={classNames(css.header)}
      size="xlarge"
      breadcrumbs={
        <Breadcrumbs
          links={[
            {
              url: routes.toARDependencyFirewallExceptions(),
              label: getString('breadcrumbs.exemptions')
            }
          ]}
        />
      }
    />
  )
}
