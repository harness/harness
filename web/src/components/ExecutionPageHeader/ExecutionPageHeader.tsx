import React, { Fragment } from 'react'
import { Layout, Text, PageHeader, Utils, Avatar, FlexExpander, Container } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color } from '@harnessio/design-system'
import { Link, useParams } from 'react-router-dom'
import { Calendar, GitFork, Timer } from 'iconoir-react'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import type { CODEProps } from 'RouteDefinitions'
import type { GitInfoProps } from 'utils/GitUtils'
import { ExecutionStatus } from 'components/ExecutionStatus/ExecutionStatus'
import { getStatus } from 'utils/ExecutionUtils'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import { timeDistance } from 'utils/Utils'
import css from './ExecutionPageHeader.module.scss'

interface BreadcrumbLink {
  label: string
  url: string
}

interface ExecutionInfo {
  message: string
  authorName: string
  authorEmail: string
  source: string
  hash: string
  status: string
  started: number
  finished: number
}

interface ExecutionPageHeaderProps extends Optional<Pick<GitInfoProps, 'repoMetadata'>, 'repoMetadata'> {
  title: string | JSX.Element
  dataTooltipId: string
  extraBreadcrumbLinks?: BreadcrumbLink[]
  executionInfo?: ExecutionInfo
}

export function ExecutionPageHeader({
  repoMetadata,
  title,
  extraBreadcrumbLinks = [],
  executionInfo
}: ExecutionPageHeaderProps) {
  const { gitRef } = useParams<CODEProps>()
  const { getString } = useStrings()
  const space = useGetSpaceParam()
  const { routes } = useAppContext()

  if (!repoMetadata) {
    return null
  }

  return (
    <PageHeader
      className={css.pageHeader}
      title={title}
      breadcrumbs={
        <Layout.Horizontal
          spacing="small"
          className={css.breadcrumb}
          padding={{ bottom: 0 }}
          margin={{ bottom: 'small' }}>
          <Link to={routes.toCODERepositories({ space })}>{getString('repositories')}</Link>
          <Icon name="main-chevron-right" size={8} color={Color.GREY_500} />
          <Link to={routes.toCODERepository({ repoPath: repoMetadata.path as string, gitRef })}>
            {repoMetadata.uid}
          </Link>
          {extraBreadcrumbLinks.map(link => (
            <Fragment key={link.url}>
              <Icon name="main-chevron-right" size={8} color={Color.GREY_500} />
              <Link to={link.url}>{link.label}</Link>
            </Fragment>
          ))}
        </Layout.Horizontal>
      }
      content={
        executionInfo && (
          <Container className={css.executionInfo}>
            <ExecutionStatus status={getStatus(executionInfo.status)} iconOnly noBackground iconSize={18} isCi />
            <Text inline color={Color.GREY_800} font={{ size: 'small' }}>
              {executionInfo.message}
            </Text>
            <PipeSeparator height={7} />
            <Avatar email={executionInfo.authorEmail} name={executionInfo.authorName} size="small" hoverCard={false} />
            <Text inline color={Color.GREY_500} font={{ size: 'small' }}>
              {executionInfo.authorName}
            </Text>
            <PipeSeparator height={7} />
            <GitFork height={12} width={12} color={Utils.getRealCSSColor(Color.GREY_500)} />
            <Text inline color={Color.GREY_500} font={{ size: 'small' }}>
              {executionInfo.source}
            </Text>
            <PipeSeparator height={7} />
            <Link
              to={routes.toCODECommit({ repoPath: repoMetadata.path as string, commitRef: executionInfo.hash })}
              className={css.hash}>
              {executionInfo.hash?.slice(0, 6)}
            </Link>
            <FlexExpander />
            <Layout.Horizontal spacing={'small'} style={{ alignItems: 'center' }} className={css.timer}>
              <Timer height={16} width={16} color={Utils.getRealCSSColor(Color.GREY_500)} />
              <Text inline color={Color.GREY_500} font={{ size: 'small' }}>
                {timeDistance(executionInfo.started, executionInfo.finished)}
              </Text>
              <PipeSeparator height={7} />
              <Calendar height={16} width={16} color={Utils.getRealCSSColor(Color.GREY_500)} />
              <Text inline color={Color.GREY_500} font={{ size: 'small' }}>
                {timeDistance(executionInfo.finished, Date.now())} ago
              </Text>
            </Layout.Horizontal>
          </Container>
        )
      }
    />
  )
}
