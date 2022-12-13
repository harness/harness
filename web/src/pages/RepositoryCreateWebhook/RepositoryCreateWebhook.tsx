import React from 'react'
import { Container, PageBody } from '@harness/uicore'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { useAppContext } from 'AppContext'
import CreateWebhookForm from './CreateWebhookForm'
import css from './RepositoryCreateWebhook.module.scss'

export default function RepositoryCreateWebhook() {
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const { repoMetadata, error, loading } = useGetRepositoryMetadata()

  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('webhook')}
        dataTooltipId="settingsWebhook"
        extraBreadcrumbLinks={
          repoMetadata && [
            {
              label: getString('settings'),
              url: routes.toCODESettings({ repoPath: repoMetadata.path as string })
            }
          ]
        }
      />
      <PageBody loading={loading} error={error}>
        {repoMetadata ? (
          <Container className={css.resourceContent}>
            <CreateWebhookForm />
          </Container>
        ) : null}
      </PageBody>
    </Container>
  )
}
