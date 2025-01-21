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
import type { FormikProps } from 'formik'
import { Expander } from '@blueprintjs/core'
import { useGetWebhookQuery, type WebhookRequest } from '@harnessio/react-har-service-client'
import { Redirect, Switch, useHistory, useParams } from 'react-router-dom'
import { Button, ButtonVariation, Container, Layout, Page, Tab, Tabs } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import RouteProvider from '@ar/components/RouteProvider/RouteProvider'
import type { RepositoryWebhookDetailsPathParams } from '@ar/routes/types'
import { useGetSpaceRef, useParentComponents, useRoutes } from '@ar/hooks'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'
import { repositoryWebhookDetailsPathParams, repositoryWebhookDetailsTabPathParams } from '@ar/routes/RouteDestinations'

import { WebhookDetailsTab } from './constants'
import WebhookDetailsTabPage from './WebhookDetailsTabPage'
// import { MOCK_WEBHOK_LIST_TABLE } from '../webhook-list/mockData'
import { WebhookDetailsContext } from './context/WebhookDetailsContext'
import { WebhookDetailsPageHeader } from './components/WebhookDetailsPageHeader/WebhookDetailsPageHeader'

import css from './WebhookDetailsPage.module.scss'

export default function WebhookDetailsPage() {
  const params = useParams<RepositoryWebhookDetailsPathParams>()
  const { repositoryIdentifier, webhookIdentifier } = params
  const history = useHistory()
  const routes = useRoutes()
  const routeDefinitions = useRoutes(true)
  const { RbacButton } = useParentComponents()

  const registryRef = useGetSpaceRef()
  const { getString } = useStrings()
  const stepRef = React.useRef<FormikProps<WebhookRequest> | null>(null)

  const [activeTab, setActiveTab] = useState(WebhookDetailsTab.Configuration)
  const [isDirty, setIsDirty] = useState(false)
  const [isUpdating, setUpdating] = useState(false)

  const { isFetching, error, data, refetch } = useGetWebhookQuery({
    registry_ref: registryRef,
    webhook_identifier: webhookIdentifier
  })

  const handleChangeTab = (nextTab: WebhookDetailsTab): void => {
    setActiveTab(nextTab)
    history.push(routes.toARRepositoryWebhookDetailsTab({ ...params, tab: nextTab }))
  }

  const renderActionBtns = (): JSX.Element => (
    <Layout.Horizontal spacing="medium">
      <RbacButton
        text={getString('save')}
        variation={ButtonVariation.PRIMARY}
        onClick={stepRef.current?.submitForm}
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
        variation={ButtonVariation.SECONDARY}
        text={getString('discard')}
        onClick={() => stepRef.current?.resetForm()}
        disabled={!isDirty}
      />
    </Layout.Horizontal>
  )

  const response = data?.content.data
  // const response = MOCK_WEBHOK_LIST_TABLE.webhooks[0]

  return (
    <WebhookDetailsContext.Provider value={{ data: response, loading: isFetching, setDirty: setIsDirty, setUpdating }}>
      <Page.Body loading={isFetching} error={error} retryOnError={() => refetch()}>
        {response && !isFetching && (
          <Container>
            <WebhookDetailsPageHeader data={response} repositoryIdentifier={repositoryIdentifier} />
            <Container className={css.tabsContainer}>
              <Tabs id="webhookDetailsTabs" selectedTabId={activeTab} onChange={handleChangeTab}>
                <Tab id={WebhookDetailsTab.Configuration} title={getString('webhookDetails.tabs.configuration')} />
                <Tab id={WebhookDetailsTab.Executions} title={getString('webhookDetails.tabs.executions')} />
                <Expander />
                {activeTab === WebhookDetailsTab.Configuration && renderActionBtns()}
              </Tabs>
            </Container>
            <Switch>
              <RouteProvider
                exact
                path={routeDefinitions.toARRepositoryWebhookDetails({ ...repositoryWebhookDetailsPathParams })}>
                <Redirect
                  to={routes.toARRepositoryWebhookDetailsTab({
                    ...params,
                    tab: WebhookDetailsTab.Configuration
                  })}
                />
              </RouteProvider>
              <RouteProvider
                exact
                path={routeDefinitions.toARRepositoryWebhookDetailsTab({ ...repositoryWebhookDetailsTabPathParams })}>
                <WebhookDetailsTabPage onInit={setActiveTab} formRef={stepRef} />
              </RouteProvider>
            </Switch>
          </Container>
        )}
      </Page.Body>
    </WebhookDetailsContext.Provider>
  )
}
