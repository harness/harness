import React from 'react'
import { PageBody, Button, Intent, Container, PageHeader } from '@harness/uicore'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'

import { RepositorySettingsHeader } from './RepositorySettingsHeader/RepositorySettingsHeader'

import emptyStateImage from './empty-state.svg'
import css from './RepositorySettings.module.scss'

export default function RepositorySettings() {
  const { repoMetadata, error, loading } = useGetRepositoryMetadata()
  const NewWebHookButton = <Button type="submit" text={'Create Webhook'} intent={Intent.PRIMARY} />
  const { getString } = useStrings()
  return (
    <Container className={css.main}>
      <PageHeader
        title=""
        breadcrumbs={repoMetadata ? <RepositorySettingsHeader repoMetadata={repoMetadata} /> : null}></PageHeader>
      <PageBody
        loading={loading}
        error={error}
        noData={{
          when: () => repoMetadata !== null,
          message: getString('noWebHooks'),
          image: emptyStateImage,
          button: NewWebHookButton
        }}>
        {repoMetadata ? (
          <>
            <RepositorySettingsHeader repoMetadata={repoMetadata} />
          </>
        ) : null}
      </PageBody>
    </Container>
  )
}
