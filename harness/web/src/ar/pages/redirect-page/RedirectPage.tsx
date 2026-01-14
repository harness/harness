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
import { Redirect } from 'react-router-dom'
import { Page, PageSpinner } from '@harnessio/uicore'

import { useRoutes } from '@ar/hooks'
import { MODULE_NAME } from '@ar/constants'
import { useParentUtils } from '@ar/hooks/useParentUtils'
import type { RedirectPageQueryParams } from '@ar/routes/types'
import RedirectWidget from '@ar/frameworks/RepositoryStep/RedirectWidget'
import { useQueryParams } from 'hooks/useQueryParams'

export default function RedirectPage() {
  const { packageType, accountId, orgIdentifier, projectIdentifier, ...rest } =
    useQueryParams<RedirectPageQueryParams>()
  const routes = useRoutes()
  const { routeToMode } = useParentUtils()

  if (routeToMode && (accountId || orgIdentifier || projectIdentifier)) {
    const queryParams = new URLSearchParams({ ...rest, packageType }).toString()
    return (
      <Redirect
        to={`${routeToMode({
          accountId,
          orgIdentifier,
          projectIdentifier,
          module: MODULE_NAME
        })}/redirect?${queryParams}`}
      />
    )
  }

  return (
    <Page.Body>
      <PageSpinner />
      <RedirectWidget packageType={packageType} />
      {!packageType && <Redirect to={routes.toARRepositories()} />}
    </Page.Body>
  )
}
