import React from 'react'
import { Container, PageBody } from '@harness/uicore'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'

import { RepositoryCreateWebhookHeader } from './RepositoryCreateWebhookHeader'
import CreateWebhookForm from './CreateWebhookForm'

import css from './RepositoryCreateWebhook.module.scss'

export default function RepositoryCreateWebhook() {
  const { repoMetadata, error, loading } = useGetRepositoryMetadata()

  return (
    <Container className={css.main}>
      <PageBody loading={loading} error={error}>
        {repoMetadata ? (
          <>
            <RepositoryCreateWebhookHeader repoMetadata={repoMetadata} />
            <Container className={css.resourceContent}>
              <CreateWebhookForm />
            </Container>
          </>
        ) : null}
      </PageBody>
    </Container>
  )
}
