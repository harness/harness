import React from 'react'
import { Container, Text, Layout, StringSubstitute } from '@harness/uicore'
import cx from 'classnames'
import ReactTimeago from 'react-timeago'
import { GitInfoProps, PullRequestState } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import type { TypesPullReq } from 'services/code'
import { PRStateLabel } from 'components/PRStateLabel/PRStateLabel'
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
      <Layout.Horizontal spacing="small" className={css.layout}>
        <PRStateLabel state={merged ? PullRequestState.MERGED : (state as PullRequestState)} />
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
