import React from 'react'
import {
  Button,
  Container,
  Color,
  Layout,
  FlexExpander,
  Text,
  FontVariation,
  Avatar,
  ButtonVariation,
  ButtonSize
} from '@harness/uicore'
import { Link } from 'react-router-dom'
import ReactTimeago from 'react-timeago'
import cx from 'classnames'
import type { RepoCommit } from 'services/scm'
import { CommitActions } from 'components/CommitActions/CommitActions'
import { useAppContext } from 'AppContext'
import { formatDate } from 'utils/Utils'
import { GitIcon, GitInfoProps } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import css from './LatestCommit.module.scss'

interface LatestCommitProps extends Pick<GitInfoProps, 'repoMetadata'> {
  latestCommit?: RepoCommit
  standaloneStyle?: boolean
}

export function LatestCommitForFolder({
  repoMetadata,
  latestCommit,
  standaloneStyle
}: LatestCommitProps): JSX.Element | null {
  const { routes } = useAppContext()
  const commitURL = routes.toSCMRepositoryCommits({
    repoPath: repoMetadata.path as string,
    commitRef: latestCommit?.sha as string
  })

  return latestCommit ? (
    <Container>
      <Layout.Horizontal spacing="small" className={cx(css.latestCommit, standaloneStyle ? css.standalone : '')}>
        <Avatar hoverCard={false} size="small" name={latestCommit.author?.identity?.name || ''} />
        <Text font={{ variation: FontVariation.SMALL_BOLD }}>
          {latestCommit.author?.identity?.name || latestCommit.author?.identity?.email}
        </Text>
        <Link to={commitURL} className={css.commitLink}>
          {latestCommit.title}
        </Link>
        <FlexExpander />
        <CommitActions sha={latestCommit.sha as string} href={commitURL} />
        <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_400}>
          <ReactTimeago date={latestCommit.author?.when as string} />
        </Text>
      </Layout.Horizontal>
    </Container>
  ) : null
}

export function LatestCommitForFile({
  repoMetadata,
  latestCommit,
  standaloneStyle
}: LatestCommitProps): JSX.Element | null {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const commitURL = routes.toSCMRepositoryCommits({
    repoPath: repoMetadata.path as string,
    commitRef: latestCommit?.sha as string
  })

  return latestCommit ? (
    <Container>
      <Layout.Horizontal
        spacing="medium"
        className={cx(css.latestCommit, css.forFile, standaloneStyle ? css.standalone : '')}>
        <Avatar hoverCard={false} size="small" name={latestCommit.author?.identity?.name || ''} />
        <Text font={{ variation: FontVariation.SMALL_BOLD }}>
          {latestCommit.author?.identity?.name || latestCommit.author?.identity?.email}
        </Text>
        <PipeSeparator />
        <Link to={commitURL} className={css.commitLink}>
          {latestCommit.title}
        </Link>
        <PipeSeparator />
        <CommitActions sha={latestCommit.sha as string} href={commitURL} />
        <PipeSeparator />
        <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_400}>
          {getString('onDate', { date: formatDate(latestCommit.author?.when as string) })}
        </Text>
        <FlexExpander />
        <Button
          size={ButtonSize.SMALL}
          icon={GitIcon.CodeHistory}
          text={getString('history')}
          variation={ButtonVariation.PRIMARY}
        />
      </Layout.Horizontal>
    </Container>
  ) : null
}

const PipeSeparator = () => <Text color={Color.GREY_200}>|</Text>
