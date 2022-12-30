import React from 'react'
import { Container, Layout } from '@harness/uicore'
import { noop } from 'lodash-es'
import type { GitInfoProps } from 'utils/GitUtils'
import { MarkdownViewer } from 'components/SourceCodeViewer/SourceCodeViewer'
import { PullRequestTabContentWrapper } from '../PullRequestTabContentWrapper'
import { PullRequestStatusInfo } from './PullRequestStatusInfo/PullRequestStatusInfo'

export const Conversation: React.FC<Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'>> = ({
  repoMetadata: _repoMetadata,
  pullRequestMetadata
}) => {
  return (
    <PullRequestTabContentWrapper loading={undefined} error={undefined} onRetry={() => noop()}>
      <Container padding="xsmall">
        <Layout.Vertical spacing="xlarge">
          <PullRequestStatusInfo />
          <Container>
            <MarkdownViewer source={pullRequestMetadata.description as string} />
          </Container>
        </Layout.Vertical>
      </Container>
    </PullRequestTabContentWrapper>
  )
}
