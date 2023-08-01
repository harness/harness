import React, { Fragment } from 'react'
import { Container, Layout, Text, Color, Icon, FontVariation, PageHeader } from '@harness/uicore'
import { Link, useParams } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import type { CODEProps } from 'RouteDefinitions'
import type { GitInfoProps } from 'utils/GitUtils'
import css from './RepositoryPageHeader.module.scss'

interface BreadcrumbLink {
  label: string
  url: string
}

interface RepositoryPageHeaderProps extends Optional<Pick<GitInfoProps, 'repoMetadata'>, 'repoMetadata'> {
  title: string | JSX.Element
  dataTooltipId: string
  extraBreadcrumbLinks?: BreadcrumbLink[]
}

export function RepositoryPageHeader({
  repoMetadata,
  title,
  dataTooltipId,
  extraBreadcrumbLinks = []
}: RepositoryPageHeaderProps) {
  const { gitRef } = useParams<CODEProps>()
  const { getString } = useStrings()
  const space = useGetSpaceParam()
  const { routes } = useAppContext()

  if (!repoMetadata) {
    return null
  }

  return (
    <PageHeader
      title=""
      breadcrumbs={
        <Container className={css.header}>
          <Layout.Horizontal spacing="small" className={css.breadcrumb}>
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
          <Container padding={{ top: 'small', bottom: 'small' }}>
            {typeof title === 'string' ? (
              <Text tag="h1" font={{ variation: FontVariation.H4 }} tooltipProps={{ dataTooltipId }}>
                {title}
              </Text>
            ) : (
              title
            )}
          </Container>
        </Container>
      }
    />
  )
}
