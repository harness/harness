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
import { HarnessDocTooltip, Page } from '@harnessio/uicore'

import { useParentComponents } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'

import css from './ManageRegistriesPage.module.scss'

function ManageRegistriesHeader() {
  const { getString } = useStrings()
  const { NGBreadcrumbs } = useParentComponents()
  return (
    <Page.Header
      className={css.pageHeader}
      title={
        <div className="ng-tooltip-native">
          <h2 data-tooltip-id="manageRegistriesPageHeading">{getString('manageRegistries.pageHeading')}</h2>
          <HarnessDocTooltip tooltipId="manageRegistriesPageHeading" useStandAlone={true} />
        </div>
      }
      breadcrumbs={<NGBreadcrumbs links={[]} />}
    />
  )
}

export default ManageRegistriesHeader
