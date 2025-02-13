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

import React, { useContext, useState } from 'react'
import type { FormikProps } from 'formik'
import { Expander } from '@blueprintjs/core'
import { Redirect, Switch, useHistory } from 'react-router-dom'
import { Button, ButtonVariation, Container, Layout, Tab, Tabs } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import type { RepositoryDetailsPathParams } from '@ar/routes/types'
import RouteProvider from '@ar/components/RouteProvider/RouteProvider'
import { useDecodedParams, useFeatureFlags, useParentComponents, useRoutes } from '@ar/hooks'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'
import { RepositoryConfigType, RepositoryPackageType } from '@ar/common/types'
import RepositoryDetailsHeaderWidget from '@ar/frameworks/RepositoryStep/RepositoryDetailsHeaderWidget'
import { repositoryDetailsPathProps, repositoryDetailsTabPathProps } from '@ar/routes/RouteDestinations'

import { RepositoryDetailsTab } from './constants'
import RepositoryDetailsTabPage from './RepositoryDetailsTabPage'
import { RepositoryProviderContext } from './context/RepositoryProvider'
import css from './RepositoryDetailsPage.module.scss'

export default function RepositoryDetails(): JSX.Element | null {
  const { RbacButton } = useParentComponents()
  const { getString } = useStrings()
  const { HAR_TRIGGERS } = useFeatureFlags()
  const pathParams = useDecodedParams<RepositoryDetailsPathParams>()
  const { repositoryIdentifier } = pathParams
  const [activeTab, setActiveTab] = useState('')
  const stepRef = React.useRef<FormikProps<unknown> | null>(null)

  const routeDefinitions = useRoutes(true)
  const history = useHistory()
  const routes = useRoutes()

  const { isDirty, data, isUpdating } = useContext(RepositoryProviderContext)

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

  const isNotUpstreamRegistry = data.config.type !== RepositoryConfigType.UPSTREAM

  return (
    <>
      <RepositoryDetailsHeaderWidget
        data={data}
        packageType={data.packageType as RepositoryPackageType}
        type={data.config.type as RepositoryConfigType}
      />
      <Container className={css.tabsContainer}>
        <Tabs id="repositoryTabDetails" selectedTabId={activeTab} onChange={handleTabChange}>
          <Tab id={RepositoryDetailsTab.PACKAGES} title={getString('repositoryDetails.tabs.packages')} />
          <Tab id={RepositoryDetailsTab.CONFIGURATION} title={getString('repositoryDetails.tabs.configuration')} />
          {HAR_TRIGGERS && isNotUpstreamRegistry && (
            <Tab id={RepositoryDetailsTab.WEBHOOKS} title={getString('repositoryDetails.tabs.webhooks')} />
          )}
          <Expander />
          {activeTab === RepositoryDetailsTab.CONFIGURATION && renderActionBtns()}
        </Tabs>
      </Container>
      <Switch>
        <RouteProvider exact path={routeDefinitions.toARRepositoryDetails({ ...repositoryDetailsPathProps })}>
          <Redirect to={routes.toARRepositoryDetailsTab({ ...pathParams, tab: RepositoryDetailsTab.CONFIGURATION })} />
        </RouteProvider>
        <RouteProvider exact path={[routeDefinitions.toARRepositoryDetailsTab({ ...repositoryDetailsTabPathProps })]}>
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
