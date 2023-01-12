import React from 'react'
import { Container, Text, Layout, Color, StringSubstitute, IconName } from '@harness/uicore'
import cx from 'classnames'
import ReactTimeago from 'react-timeago'
import { CodeIcon, GitInfoProps, PullRequestState } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import type { TypesPullReq } from 'services/code'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import { GitRefLink } from 'components/GitRefLink/GitRefLink'
import css from './PullRequestMetaLine.module.scss'

export const PullRequestMetaLine: React.FC<TypesPullReq & Pick<GitInfoProps, 'repoMetadata'>> = ({
  repoMetadata,
  target_branch,
  source_branch,
  author,
  edited,
  merged,
  state
}) => {
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const vars = {
    user: <strong>{author?.display_name}</strong>,
    number: <strong>5</strong>, // TODO: No data from backend now
    target: (
      <GitRefLink
        text={target_branch as string}
        url={routes.toCODERepository({ repoPath: repoMetadata.path as string, gitRef: target_branch })}
      />
    ),
    source: (
      <GitRefLink
        text={source_branch as string}
        url={routes.toCODERepository({ repoPath: repoMetadata.path as string, gitRef: source_branch })}
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
          <ReactTimeago date={edited as number} />
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
