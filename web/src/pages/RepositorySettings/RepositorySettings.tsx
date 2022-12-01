import React from 'react'
import { useHistory } from 'react-router-dom'

import { PageBody, Button, Intent, Container } from '@harness/uicore'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { CodeIcon } from 'utils/GitUtils'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { getErrorMessage } from 'utils/Utils'
import emptyStateImage from 'images/empty-state.svg'
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
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('settings')}
        dataTooltipId="repositorySettings"
      />
      <PageBody
        loading={loading}
        error={getErrorMessage(error)}
        noData={{
          when: () => !hooks.length,
          message: getString('noWebHooks'),
          image: emptyStateImage,
          button: NewWebHookButton
        }}>
        {repoMetadata ? <></> : null}
      </PageBody>
    </Container>
  )
}
