import React from 'react'
import { Container } from '@harness/uicore'
import type { GitInfoProps } from 'utils/GitUtils'
import { MarkdownViewer } from 'components/SourceCodeViewer/SourceCodeViewer'
import { PullRequestTabContentWrapper } from '../PullRequestTabContentWrapper'

export const PullRequestConversation: React.FC<Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'>> = ({
  repoMetadata,
  pullRequestMetadata
}) => {
  return (
    <PullRequestTabContentWrapper loading={undefined} error={undefined} onRetry={() => {}}>
      <Container>
        <MarkdownViewer source={pullRequestMetadata.description} />
      </Container>
    </PullRequestTabContentWrapper>
  )
}
