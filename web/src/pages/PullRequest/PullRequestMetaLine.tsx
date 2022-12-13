import React from 'react'
import { Container, Text, Layout, Color, StringSubstitute, IconName } from '@harness/uicore'
import cx from 'classnames'
import ReactTimeago from 'react-timeago'
import { CodeIcon, GitInfoProps, PullRequestState } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import { GitRefLink } from 'components/GitRefLink/GitRefLink'
import type { PullRequestResponse } from 'utils/types'
import css from './PullRequestMetaLine.module.scss'

export const PullRequestMetaLine: React.FC<PullRequestResponse & Pick<GitInfoProps, 'repoMetadata'>> = ({
  repoMetadata,
  targetBranch,
  sourceBranch,
  createdBy = '',
  updated,
  merged,
  state
}) => {
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const vars = {
    user: <strong>{createdBy}</strong>,
    number: <strong>5</strong>, // TODO: No data from backend now
    target: (
      <GitRefLink
        text={targetBranch}
        url={routes.toCODERepository({ repoPath: repoMetadata.path as string, gitRef: targetBranch })}
      />
    ),
    source: (
      <GitRefLink
        text={sourceBranch}
        url={routes.toCODERepository({ repoPath: repoMetadata.path as string, gitRef: sourceBranch })}
      />
    )
  }

  return (
    <Container padding={{ left: 'xlarge' }} className={css.main}>
      <Layout.Horizontal spacing="small">
        <PullRequestStateLabel state={merged ? PullRequestState.MERGED : (state as PullRequestState)} />
        <Text className={css.metaline}>
          <StringSubstitute str={getString('pr.metaLine')} vars={vars} />
        </Text>
        <PipeSeparator height={9} />
        <Text inline className={cx(css.metaline, css.time)}>
          <ReactTimeago date={updated} />
        </Text>
      </Layout.Horizontal>
    </Container>
  )
}

const PullRequestStateLabel: React.FC<{ state: PullRequestState }> = ({ state }) => {
  const { getString } = useStrings()

  let color = Color.GREEN_700
  let icon: IconName = CodeIcon.PullRequest
  let clazz: typeof css | string = ''

  switch (state) {
    case PullRequestState.MERGED:
      color = Color.PURPLE_700
      icon = CodeIcon.PullRequest
      clazz = css.merged
      break
    case PullRequestState.CLOSED:
      color = Color.GREY_600
      icon = CodeIcon.PullRequest
      clazz = css.closed
      break
    case PullRequestState.REJECTED:
      color = Color.RED_600
      icon = CodeIcon.PullRequestRejected
      clazz = css.rejected
      break
  }

  return (
    <Text className={cx(css.state, clazz)} icon={icon} iconProps={{ color, size: 9 }}>
      <StringSubstitute str={getString('pr.state')} vars={{ state }} />
    </Text>
  )
}
