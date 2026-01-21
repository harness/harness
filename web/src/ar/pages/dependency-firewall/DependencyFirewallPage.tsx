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

import React, { useState } from 'react'
import { Redirect, Switch, useHistory } from 'react-router-dom'
import { HarnessDocTooltip, Page, Tab, Tabs } from '@harnessio/uicore'

import { useRoutes } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import Breadcrumbs from '@ar/components/Breadcrumbs/Breadcrumbs'
import TabsContainer from '@ar/components/TabsContainer/TabsContainer'
import RouteProvider from '@ar/components/RouteProvider/RouteProvider'

import { DependencyFirewallTab, DependencyFirewallTabs } from './constants'

import css from './DependencyFirewallPage.module.scss'

const DependencyFirewallPage: React.FC = () => {
  const { getString } = useStrings()
  const [activeTab, setActiveTab] = useState<DependencyFirewallTab | undefined>()
  const history = useHistory()
  const routes = useRoutes()
  const routeDefinitions = useRoutes(true)

  const handleTabChange = (nextTab: DependencyFirewallTab): void => {
    setActiveTab(nextTab)
    switch (nextTab) {
      case DependencyFirewallTab.VIOLATIONS:
        history.push(routes.toARDependencyFirewallViolations())
        break
      case DependencyFirewallTab.EXCEPTIONS:
        history.push(routes.toARDependencyFirewallExceptions())
        break
    }
  }

  return (
    <>
      <Page.Header
        className={css.pageHeader}
        title={
          <div className="ng-tooltip-native">
            <h2 data-tooltip-id="dependencyFirewallPageHeading">{getString('dependencyFirewall.pageHeading')}</h2>
            <HarnessDocTooltip tooltipId="dependencyFirewallPageHeading" useStandAlone={true} />
          </div>
        }
        breadcrumbs={<Breadcrumbs links={[]} />}
      />
      <TabsContainer className={css.tabsContainer}>
        <Tabs id="dependencyFirewallTabDetails" selectedTabId={activeTab} onChange={handleTabChange}>
          {DependencyFirewallTabs.map(each => (
            <Tab key={each.value} id={each.value} title={getString(each.label)} disabled={each.disabled} />
          ))}
        </Tabs>
      </TabsContainer>
      <Switch>
        <RouteProvider exact path={routeDefinitions.toARDependencyFirewall()}>
          <Redirect to={routes.toARDependencyFirewallViolations()} />
        </RouteProvider>
        <RouteProvider
          exact
          path={routeDefinitions.toARDependencyFirewallViolations()}
          onLoad={() => {
            setActiveTab(DependencyFirewallTab.VIOLATIONS)
          }}>
          <div>Violations</div>
        </RouteProvider>
        <RouteProvider
          exact
          path={routeDefinitions.toARDependencyFirewallExceptions()}
          onLoad={() => {
            setActiveTab(DependencyFirewallTab.EXCEPTIONS)
          }}>
          <div>Exceptions (Coming soon)</div>
        </RouteProvider>
      </Switch>
    </>
  )
}

export default DependencyFirewallPage
