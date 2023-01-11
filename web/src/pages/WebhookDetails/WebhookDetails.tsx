import React from 'react'
import { Container, PageBody } from '@harness/uicore'
import { useGet } from 'restful-react'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import type { OpenapiWebhookType } from 'services/code'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { WehookForm } from 'pages/WebhookNew/WehookForm'
import { useAppContext } from 'AppContext'

export default function WebhookDetails() {
  const { getString } = useStrings()
  const { routes } = useAppContext()
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
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('webhookDetails')}
        dataTooltipId="webhookDetails"
        extraBreadcrumbLinks={
          repoMetadata && [
            {
              label: getString('webhooks'),
              url: routes.toCODEWebhooks({ repoPath: repoMetadata.path as string })
            }
          ]
        }
      />
      <PageBody
        loading={loading || webhookLoading}
        error={error || webhookError}
        retryOnError={() => (repoMetadata ? refetchWebhook() : refreshMetadata())}>
        {repoMetadata && data && <WehookForm isEdit webhook={data} repoMetadata={repoMetadata} />}
      </PageBody>
    </Container>
  )
}
