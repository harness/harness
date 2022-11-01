import React from 'react'
import {
  Container,
  Color,
  Layout,
  Button,
  ButtonSize,
  FlexExpander,
  ButtonVariation,
  Text,
  FontVariation
} from '@harness/uicore'
import { Link } from 'react-router-dom'
import ReactTimeago from 'react-timeago'
import cx from 'classnames'
import type { RepoCommit } from 'services/scm'
import css from './LatestCommit.module.scss'

interface LatestCommitProps {
  latestCommit?: RepoCommit
  standaloneStyle?: boolean
}

export function LatestCommit({ latestCommit, standaloneStyle }: LatestCommitProps): JSX.Element | null {
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
        <Button
          className={css.shaBtn}
          text={latestCommit.sha?.substring(0, 6)}
          variation={ButtonVariation.SECONDARY}
          size={ButtonSize.SMALL}
        />
        <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_400}>
          <ReactTimeago date={latestCommit.author?.when as string} />
        </Text>
      </Layout.Horizontal>
    </Container>
  ) : null
}
