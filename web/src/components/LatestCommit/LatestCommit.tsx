import React from 'react'
import { Container, Color, Layout, FlexExpander, Text, FontVariation, Avatar } from '@harness/uicore'
import { Link } from 'react-router-dom'
import { Render } from 'react-jsx-match'
import ReactTimeago from 'react-timeago'
import cx from 'classnames'
import type { TypesCommit } from 'services/code'
import { CommitActions } from 'components/CommitActions/CommitActions'
import { useAppContext } from 'AppContext'
import { formatDate } from 'utils/Utils'
import type { GitInfoProps } from 'utils/GitUtils'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import css from './LatestCommit.module.scss'

interface LatestCommitProps extends Pick<GitInfoProps, 'repoMetadata'> {
  latestCommit?: TypesCommit
  standaloneStyle?: boolean
}

export function LatestCommitForFolder({ repoMetadata, latestCommit, standaloneStyle }: LatestCommitProps) {
  const { routes } = useAppContext()
  const commitURL = routes.toCODECommits({
    repoPath: repoMetadata.path as string,
    commitRef: latestCommit?.sha as string
  })

  return (
    <Render when={latestCommit}>
      <Container>
        <Layout.Horizontal spacing="small" className={cx(css.latestCommit, { [css.standalone]: standaloneStyle })}>
          <Avatar hoverCard={false} size="small" name={latestCommit?.author?.identity?.name || ''} />
          <Text font={{ variation: FontVariation.SMALL_BOLD }}>
            {latestCommit?.author?.identity?.name || latestCommit?.author?.identity?.email}
          </Text>
          <Link to={commitURL} className={css.commitLink}>
            {latestCommit?.title}
          </Link>
          <FlexExpander />
          <CommitActions sha={latestCommit?.sha as string} href={commitURL} />
          <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_400}>
            <ReactTimeago date={latestCommit?.author?.when as string} />
          </Text>
        </Layout.Horizontal>
      </Container>
    </Render>
  )
}

export function LatestCommitForFile({ repoMetadata, latestCommit, standaloneStyle }: LatestCommitProps) {
  const { routes } = useAppContext()
  const commitURL = routes.toCODECommits({
    repoPath: repoMetadata.path as string,
    commitRef: latestCommit?.sha as string
  })

  return (
    <Render when={latestCommit}>
      <Container>
        <Layout.Horizontal
          spacing="medium"
          className={cx(css.latestCommit, css.forFile, { [css.standalone]: standaloneStyle })}>
          <Avatar hoverCard={false} size="small" name={latestCommit?.author?.identity?.name || ''} />
          <Text font={{ variation: FontVariation.SMALL_BOLD }}>
            {latestCommit?.author?.identity?.name || latestCommit?.author?.identity?.email}
          </Text>
          <PipeSeparator height={9} />

          <Link to={commitURL} className={css.commitLink}>
            {latestCommit?.title}
          </Link>
          <PipeSeparator height={9} />
          <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_400}>
            {formatDate(latestCommit?.author?.when as string)}
          </Text>

          <FlexExpander />
          <CommitActions sha={latestCommit?.sha as string} href={commitURL} />
        </Layout.Horizontal>
      </Container>
    </Render>
  )
}
