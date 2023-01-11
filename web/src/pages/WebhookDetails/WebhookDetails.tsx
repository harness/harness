import React from 'react'
import { Container, PageBody } from '@harness/uicore'
import { useGet } from 'restful-react'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import type { OpenapiWebhookType } from 'services/code'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { WehookForm } from 'pages/WebhookNew/WehookForm'

export default function WebhookDetails() {
  const { getString } = useStrings()
  const { repoMetadata, error, loading, webhookId } = useGetRepositoryMetadata()
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
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('webhookDetails')}
        dataTooltipId="webhookDetails"
      />
      <PageBody
        loading={loading || webhookLoading}
        error={error || webhookError}
        retryOnError={() => {
          refetchWebhook()
        }}>
        {repoMetadata && data && <WehookForm isEdit webhook={data} repoMetadata={repoMetadata} />}
      </PageBody>
    </Container>
  )
}
