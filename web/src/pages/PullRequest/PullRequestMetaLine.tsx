import React from 'react'
import { Container, Text, Layout, StringSubstitute } from '@harness/uicore'
import cx from 'classnames'
import ReactTimeago from 'react-timeago'
import { CodeIcon, type GitInfoProps } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import type { TypesPullReq } from 'services/code'
import { PullRequestStateLabel } from 'components/PullRequestStateLabel/PullRequestStateLabel'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import { GitRefLink } from 'components/GitRefLink/GitRefLink'
import css from './PullRequestMetaLine.module.scss'
import BranchCopyButton from 'components/BranchCopyButton/BranchCopyButton'
import { CopyButton } from 'components/CopyButton/CopyButton'

export const PullRequestMetaLine: React.FC<TypesPullReq & Pick<GitInfoProps, 'repoMetadata'>> = ({
  repoMetadata,
  target_branch,
  source_branch,
  author,
  edited,
  state,
  is_draft,
  stats
}) => {
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const vars = {
    user: <strong>{author?.display_name}</strong>,
    commits: <strong>{stats?.commits}</strong>,
    commitsCount: stats?.commits,
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
      <Layout.Horizontal spacing="small" className={css.layout}>
        <PullRequestStateLabel data={{ is_draft, state }} />
        <Text className={css.metaline}>
          <StringSubstitute str={getString('pr.metaLine')} vars={vars} />
          {source_branch ? (
            <CopyButton
              content={source_branch}
              tooltip={getString('copyBranch')}
              icon={CodeIcon.Copy}
              iconProps={{ size: 14 }}
              padding={{ bottom: 'xsmall' }}
            />
          ) : null}
        </Text>

        <PipeSeparator height={9} />
        <Text inline className={cx(css.metaline, css.time)}>
          <ReactTimeago date={edited as number} />
        </Text>
      </Layout.Horizontal>
    </Container>
  )
}
