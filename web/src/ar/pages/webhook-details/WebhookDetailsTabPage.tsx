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

import React, { useEffect } from 'react'
import type { FormikProps } from 'formik'
import { useParams } from 'react-router-dom'
import type { WebhookRequest } from '@harnessio/react-har-service-client'

import type { RepositoryWebhookDetailsTabPathParams } from '@ar/routes/types'

import { WebhookDetailsTab } from './constants'
import WebhookConfigurationForm from './components/WebhookConfigurationForm/WebhookConfigurationForm'

interface WebhookDetailsTabPageProps {
  onInit: (tab: WebhookDetailsTab) => void
  formRef: React.RefObject<FormikProps<WebhookRequest>>
}

export default function WebhookDetailsTabPage(props: WebhookDetailsTabPageProps): JSX.Element {
  const { onInit, formRef } = props
  const params = useParams<RepositoryWebhookDetailsTabPathParams>()
  const { tab } = params

  useEffect(() => {
    onInit(tab)
  }, [tab])

  switch (tab) {
    case WebhookDetailsTab.Configuration:
      return <WebhookConfigurationForm formRef={formRef} />
    case WebhookDetailsTab.Executions:
      return <>Executions Page</>
    default:
      return <>Not Found</>
  }
}
