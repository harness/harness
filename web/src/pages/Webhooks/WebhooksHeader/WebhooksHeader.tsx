import { useHistory } from 'react-router-dom'
import React from 'react'
import { Container, Layout, FlexExpander, ButtonVariation, Button } from '@harness/uicore'
import { useStrings } from 'framework/strings'
import { CodeIcon, GitInfoProps } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
import css from './WebhooksHeader.module.scss'

export function WebhooksHeader({ repoMetadata }: Pick<GitInfoProps, 'repoMetadata'>) {
  const history = useHistory()
  const { routes } = useAppContext()
  const { getString } = useStrings()

  return (
    <Container className={css.main} padding="xlarge">
      <Layout.Horizontal spacing="medium">
        <FlexExpander />
        <Button
          variation={ButtonVariation.PRIMARY}
          text={getString('createWebhook')}
          icon={CodeIcon.Add}
          onClick={() => history.push(routes.toCODEWebhookNew({ repoPath: repoMetadata?.path as string }))}
        />
      </Layout.Horizontal>
    </Container>
  )
}
