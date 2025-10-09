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
import { Tab } from '@blueprintjs/core'
import { Tabs, Text } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { Redirect, Switch, useHistory } from 'react-router-dom'

import { useRoutes } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import TabsContainer from '@ar/components/TabsContainer/TabsContainer'
import RouteProvider from '@ar/components/RouteProvider/RouteProvider'
import { manageRegistriesTabPathProps } from '@ar/routes/RouteDestinations'

import ManageRegistriesTabs from './ManageRegistriesTabs'
import { ManageRegistriesDetailsTab, ManageRegistriesDetailsTabs } from './constants'
import css from './ManageRegistriesPage.module.scss'

function ManageRegistriesDetails() {
  const { getString } = useStrings()
  const routeDefinitions = useRoutes(true)
  const routes = useRoutes()
  const history = useHistory()
  const [activeTab, setActiveTab] = useState('')

  const handleTabChange = (nextTab: ManageRegistriesDetailsTab): void => {
    history.push(routes.toARManageRegistriesTab({ tab: nextTab }))
    setActiveTab(nextTab)
  }

  return (
    <>
      <TabsContainer className={css.tabsContainer}>
        <Tabs id="manageRegistriesDetailsTabs" selectedTabId={activeTab} onChange={handleTabChange}>
          {ManageRegistriesDetailsTabs.map(each => (
            <Tab
              key={each.value}
              id={each.value}
              disabled={each.disabled}
              title={
                <Text
                  font={{ variation: FontVariation.BODY, weight: activeTab === each.value ? 'bold' : 'light' }}
                  tooltip={each.tooltip ? getString(each.tooltip) : undefined}>
                  {getString(each.label)}
                </Text>
              }
            />
          ))}
        </Tabs>
      </TabsContainer>
      <Switch>
        <RouteProvider exact path={routeDefinitions.toARManageRegistries()}>
          <Redirect to={routes.toARManageRegistriesTab({ tab: ManageRegistriesDetailsTab.LABELS })} />
        </RouteProvider>
        <RouteProvider
          exact
          path={[routeDefinitions.toARManageRegistriesTab({ ...manageRegistriesTabPathProps })]}
          onLoad={({ tab }) => {
            setActiveTab(tab)
          }}>
          <ManageRegistriesTabs />
        </RouteProvider>
      </Switch>
    </>
  )
}

export default ManageRegistriesDetails
