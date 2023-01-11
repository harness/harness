import React from 'react'
import { Container, PageBody } from '@harness/uicore'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { useAppContext } from 'AppContext'
import { WehookForm } from './WehookForm'

export default function WebhookNew() {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const { repoMetadata, error, loading } = useGetRepositoryMetadata()

  return (
    <Container>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('createWebhook')}
        dataTooltipId="webhooks"
        extraBreadcrumbLinks={
          repoMetadata && [
            {
              label: getString('webhooks'),
              url: routes.toCODEWebhooks({ repoPath: repoMetadata.path as string })
            }
          ]
        }
      />
      <PageBody loading={loading} error={error}>
        {repoMetadata && <WehookForm repoMetadata={repoMetadata} />}
      </PageBody>
    </Container>
  )
}
