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

import React, { useEffect } from 'react'
import cx from 'classnames'
import { PageBody, Container, Tabs } from '@harnessio/uicore'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { PageBrowserProps, getErrorMessage, voidFn } from 'utils/Utils'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { WebhookTabs } from 'utils/GitUtils'
import WebhookDetailsTab from 'pages/WebhookDetailsTab/WebhookDetailsTab'
import WebhookExecutions from 'pages/WebhookExecutions/WebhookExecutions'
import { useUpdateQueryParams } from 'hooks/useUpdateQueryParams'
import { useQueryParams } from 'hooks/useQueryParams'
import css from './Webhook.module.scss'

export default function WebhookDetails() {
  const { repoMetadata, error, loading, refetch, webhookId } = useGetRepositoryMetadata()
  const queryParams = useQueryParams<PageBrowserProps>()
  const { replaceQueryParams } = useUpdateQueryParams()
  const { getString } = useStrings()

  useEffect(() => {
    if (!queryParams.tab) {
      replaceQueryParams({ ...queryParams, tab: WebhookTabs.DETAILS })
    }
  })

  const tabListArray = [
    {
      id: WebhookTabs.DETAILS,
      title: getString('details'),
      panel: (
        <Container padding={'large'}>
          <WebhookDetailsTab />
        </Container>
      )
    },
    {
      id: WebhookTabs.EXECUTIONS,
      title: getString('pageTitle.executions'),
      panel: <WebhookExecutions />
    }
  ]
  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        className={css.headerContainer}
        repoMetadata={repoMetadata}
        title={`${getString('webhook')} : ${webhookId}`}
        dataTooltipId={getString('webhookPage')}
      />
      <PageBody error={getErrorMessage(error)} retryOnError={voidFn(refetch)}>
        <LoadingSpinner visible={loading} />
        {repoMetadata && (
          <Container className={cx(css.main, css.tabsContainer)}>
            <Tabs
              id={getString('webhookTabs')}
              large={false}
              selectedTabId={queryParams.tab}
              animate={false}
              onChange={(id: WebhookTabs) => {
                if (id === WebhookTabs.DETAILS) {
                  delete queryParams.page
                }
                replaceQueryParams({ ...queryParams, tab: id })
              }}
              tabList={tabListArray}></Tabs>
          </Container>
        )}
      </PageBody>
    </Container>
  )
}
