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
import { Container } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings/String'
import { useRoutes } from '@ar/hooks'

import SideNavHeader from './components/SideNavHeader/SideNavHeader'
import SideNavLink from './components/SideNavLink/SideNavLink'
import { SideNavSection } from './components/SideNavSection/SideNavSection'

import css from './SideNav.module.scss'

export function SideNav(): JSX.Element {
  const { getString } = useStrings()
  const routes = useRoutes()
  return (
    <Container className={classNames(css.container, css.expanded)}>
      <SideNavHeader />
      <Container>
        <SideNavSection>
          <SideNavLink icon="nav-pipeline" label={getString('sideNav.repositories')} to={routes.toARRepositories()} />
          <SideNav.Link icon="ssca-artifacts" label={getString('sideNav.artifacts')} to={routes.toARArtifacts()} />
        </SideNavSection>
      </Container>
    </Container>
  )
}

SideNav.Link = SideNavLink
SideNav.Section = SideNavSection
