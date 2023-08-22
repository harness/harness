import React from 'react'
import { Container, PageBody } from '@harnessio/uicore'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
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
        title={getString('createAWebhook')}
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
      <PageBody error={error}>
        <LoadingSpinner visible={loading} />
        {repoMetadata && <WehookForm repoMetadata={repoMetadata} />}
      </PageBody>
    </Container>
  )
}
