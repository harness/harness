/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react'
import cx from 'classnames'
import { Container, Page } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import { SpacePullRequestsListing } from 'pages/PullRequests/SpacePullRequestsListing'
import { PRFilterProvider } from 'contexts/PRFiltersContext'
import css from '../ManageSpace.module.scss'

export default function SpacePullRequests() {
  const { getString } = useStrings()

  return (
    <Container className={css.main}>
      <Page.Header title={getString('pullRequests')} />

      <Container className={cx(css.main, css.tabsContainer)}>
        <PRFilterProvider>
          <SpacePullRequestsListing />
        </PRFilterProvider>
      </Container>
    </Container>
  )
}
