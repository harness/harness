import React from 'react'
import { Container, Color, Layout, FlexExpander, Text, FontVariation } from '@harness/uicore'
import { Link } from 'react-router-dom'
import ReactTimeago from 'react-timeago'
import cx from 'classnames'
import type { RepoCommit, TypesRepository } from 'services/scm'
import { CommitActions } from 'components/CommitActions/CommitActions'
import { useAppContext } from 'AppContext'
import css from './LatestCommit.module.scss'

interface LatestCommitProps {
  repoMetadata: TypesRepository
  latestCommit?: RepoCommit
  standaloneStyle?: boolean
}

export function LatestCommit({ repoMetadata, latestCommit, standaloneStyle }: LatestCommitProps): JSX.Element | null {
  const { routes } = useAppContext()

  return latestCommit ? (
    <Container>
      <Layout.Horizontal spacing="small" className={cx(css.latestCommit, standaloneStyle ? css.standalone : '')}>
        <Text font={{ variation: FontVariation.SMALL_BOLD }}>
          {latestCommit.author?.identity?.name || latestCommit.author?.identity?.email}
        </Text>
        <Link to="" className={css.commitLink}>
          {latestCommit.title}
        </Link>
        <FlexExpander />
        <CommitActions
          sha={latestCommit.sha as string}
          href={routes.toSCMRepositoryCommits({
            repoPath: repoMetadata.path as string,
            commitRef: latestCommit.sha as string
          })}
        />
        <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_400}>
          <ReactTimeago date={latestCommit.author?.when as string} />
        </Text>
      </Layout.Horizontal>
    </Container>
  ) : null
}
