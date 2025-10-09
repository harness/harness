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
import { Container, PageBody } from '@harnessio/uicore'
import { useGet } from 'restful-react'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import type { OpenapiWebhookType } from 'services/code'
import { WehookForm } from 'pages/WebhookNew/WehookForm'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'

export default function WebhookDetails() {
  const { repoMetadata, error, loading, webhookId, refetch: refreshMetadata } = useGetRepositoryMetadata()
  const {
    data,
    loading: webhookLoading,
    error: webhookError,
    refetch: refetchWebhook
  } = useGet<OpenapiWebhookType>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/webhooks/${webhookId}`,
    lazy: !repoMetadata
  })

  return (
    <Container>
      <PageBody
        error={error || webhookError}
        retryOnError={() => (repoMetadata ? refetchWebhook() : refreshMetadata())}>
        <LoadingSpinner visible={loading || webhookLoading} withBorder={!!data && webhookLoading} />
        {repoMetadata && data && <WehookForm isEdit webhook={data} repoMetadata={repoMetadata} />}
      </PageBody>
    </Container>
  )
}
