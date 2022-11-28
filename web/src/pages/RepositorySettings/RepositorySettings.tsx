import React from 'react'
import { useHistory } from 'react-router-dom'

import { PageBody, Button, Intent, Container, PageHeader } from '@harness/uicore'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { CodeIcon } from 'utils/GitUtils'

import { RepositorySettingsHeader } from './RepositorySettingsHeader/RepositorySettingsHeader'

import emptyStateImage from './empty-state.svg'

import css from './RepositorySettings.module.scss'

export default function RepositorySettings() {
  const { repoMetadata, error, loading } = useGetRepositoryMetadata()
  const { routes } = useAppContext()
  const history = useHistory()

  const NewWebHookButton = (
    <Button
      type="button"
      text={'Create Webhook'}
      intent={Intent.PRIMARY}
      icon={CodeIcon.Add}
      onClick={() => {
        history.push(
          routes.toCODECreateWebhook({
            repoPath: repoMetadata?.path as string
          })
        )
      }}
    />
  )
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
