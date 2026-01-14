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

import React, { useContext, useEffect, useMemo, useState } from 'react'
import type { FormikProps } from 'formik'
import { Expander } from '@blueprintjs/core'
import { Redirect, Switch, useHistory } from 'react-router-dom'
import { Button, ButtonVariation, Layout, Tab, Tabs } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import type { RepositoryDetailsPathParams } from '@ar/routes/types'
import RouteProvider from '@ar/components/RouteProvider/RouteProvider'
import TabsContainer from '@ar/components/TabsContainer/TabsContainer'
import {
  useAppStore,
  useDecodedParams,
  useFeatureFlags,
  useGetRepositoryListViewType,
  useParentComponents,
  useRoutes
} from '@ar/hooks'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'
import repositoryFactory from '@ar/frameworks/RepositoryStep/RepositoryFactory'
import { repositoryDetailsPathProps, repositoryDetailsTabPathProps } from '@ar/routes/RouteDestinations'

import { RepositoryDetailsTab, RepositoryDetailsTabs } from './constants'
import RepositoryDetailsTabPage from './RepositoryDetailsTabPage'
import { RepositoryProviderContext } from './context/RepositoryProvider'
import css from './RepositoryDetailsPage.module.scss'

export default function RepositoryDetails(): JSX.Element | null {
  const { RbacButton } = useParentComponents()
  const { getString } = useStrings()
  const featureFlags = useFeatureFlags()
  const pathParams = useDecodedParams<RepositoryDetailsPathParams>()
  const { repositoryIdentifier } = pathParams
  const [activeTab, setActiveTab] = useState('')
  const stepRef = React.useRef<FormikProps<unknown> | null>(null)

  const routeDefinitions = useRoutes(true)
  const { isCurrentSessionPublic } = useAppStore()
  const history = useHistory()
  const routes = useRoutes()
  const repositoryListViewType = useGetRepositoryListViewType()

  const { isDirty, data, isUpdating } = useContext(RepositoryProviderContext)

  const repositoryTabs = useMemo(() => {
    if (!data) return []
    const repositoryType = repositoryFactory.getRepositoryType(data?.packageType)
    const tabs = repositoryType?.getSupportedRepositoryTabs() || []
    return RepositoryDetailsTabs.filter(each => tabs.includes(each.value))
      .filter(each => !each.featureFlag || featureFlags[each.featureFlag])
      .filter(each => !each.type || each.type === data?.config.type)
      .filter(each => !each.mode || each.mode === repositoryListViewType)
      .filter(each => (isCurrentSessionPublic ? each.isSupportedInPublicView : true))
  }, [data, featureFlags, repositoryListViewType, isCurrentSessionPublic])

  const handleTabChange = (nextTab: RepositoryDetailsTab): void => {
    setActiveTab(nextTab)
    history.push(routes.toARRepositoryDetailsTab({ ...pathParams, tab: nextTab }))
  }

  const handleSubmitForm = (): void => {
    stepRef.current?.submitForm()
  }

  const handleResetForm = (): void => {
    stepRef.current?.resetForm()
  }

  useEffect(() => {
    if (!isCurrentSessionPublic) return
    if (repositoryTabs.length === 0) return
    if (repositoryTabs.find(each => each.value === activeTab)) return
    handleTabChange(repositoryTabs[0].value)
  }, [activeTab, repositoryTabs, isCurrentSessionPublic])

  const renderActionBtns = (): JSX.Element => (
    <Layout.Horizontal className={css.btnContainer}>
      <RbacButton
        text={getString('save')}
        className={css.saveButton}
        variation={ButtonVariation.PRIMARY}
        onClick={handleSubmitForm}
        disabled={!isDirty || isUpdating}
        permission={{
          permission: PermissionIdentifier.EDIT_ARTIFACT_REGISTRY,
          resource: {
            resourceType: ResourceType.ARTIFACT_REGISTRY,
            resourceIdentifier: repositoryIdentifier
          }
        }}
      />
      <Button
        className={css.discardBtn}
        variation={ButtonVariation.SECONDARY}
        text={getString('discard')}
        onClick={handleResetForm}
        disabled={!isDirty}
      />
    </Layout.Horizontal>
  )

  if (!data) return null

  return (
    <>
      <TabsContainer className={css.tabsContainer}>
        <Tabs id="repositoryTabDetails" selectedTabId={activeTab} onChange={handleTabChange}>
          {repositoryTabs.map(each => (
            <Tab key={each.value} id={each.value} title={getString(each.label)} />
          ))}
          <Expander />
          {activeTab === RepositoryDetailsTab.CONFIGURATION && renderActionBtns()}
        </Tabs>
      </TabsContainer>
      <Switch>
        <RouteProvider isPublic exact path={routeDefinitions.toARRepositoryDetails({ ...repositoryDetailsPathProps })}>
          <Redirect to={routes.toARRepositoryDetailsTab({ ...pathParams, tab: repositoryTabs[0].value })} />
        </RouteProvider>
        <RouteProvider
          isPublic
          exact
          path={[routeDefinitions.toARRepositoryDetailsTab({ ...repositoryDetailsTabPathProps })]}>
          <RepositoryDetailsTabPage
            onInit={nextTab => {
              setActiveTab(nextTab)
            }}
            stepRef={stepRef}
          />
        </RouteProvider>
      </Switch>
    </>
  )
}
