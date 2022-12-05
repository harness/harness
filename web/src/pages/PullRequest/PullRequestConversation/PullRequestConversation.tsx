import React from 'react'
import { Container } from '@harness/uicore'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import type { GitInfoProps } from 'utils/GitUtils'
import css from './PullRequestConversation.module.scss'

export const PullRequestConversation: React.FC<Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'>> = ({
  repoMetadata,
  pullRequestMetadata
}) => {
  const { getString } = useStrings()
  const { routes } = useAppContext()

  return (
    <Container className={css.main} padding="xlarge">
      CONVERSATION...
    </Container>
  )
}
