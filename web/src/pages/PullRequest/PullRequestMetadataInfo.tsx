import React from 'react'
import { Container, Text, Layout, Color } from '@harness/uicore'
import cx from 'classnames'
import ReactTimeago from 'react-timeago'
import { CodeIcon, GitInfoProps } from 'utils/GitUtils'
// import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import { StringSubstitute } from 'components/StringSubstitute/StringSubstitute'
import { GitRefLink } from 'components/GitRefLink/GitRefLink'
import type { PullRequestResponse } from 'utils/types'
import css from './PullRequestMetadataInfo.module.scss'

export const PullRequestMetadataInfo: React.FC<PullRequestResponse & Pick<GitInfoProps, 'repoMetadata'>> = ({
  createdBy = '',
  targetBranch,
  sourceBranch,
  updated
  // repoMetadata
}) => {
  const { getString } = useStrings()
  // const { routes } = useAppContext()
  const vars = {
    user: <strong>{createdBy}</strong>,
    number: 1, // TODO: No data from backend now
    target: <GitRefLink text={targetBranch} url="TODO" />,
    source: <GitRefLink text={sourceBranch} url="TODO" />
  }

  return (
    <Container padding={{ left: 'xlarge' }} className={css.main}>
      <Layout.Horizontal spacing="small">
        <Text className={css.state} icon={CodeIcon.PullRequest} iconProps={{ color: Color.GREEN_700, size: 9 }}>
          Open
        </Text>
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
