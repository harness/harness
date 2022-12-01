import React from 'react'
import { Container, Layout, Text, Color, Icon, FontVariation, PageHeader } from '@harness/uicore'
import { Link } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import type { GitInfoProps } from 'utils/GitUtils'
import css from './RepositoryPageHeader.module.scss'

interface RepositoryPageHeaderProps extends Optional<Pick<GitInfoProps, 'repoMetadata'>, 'repoMetadata'> {
  title: string | JSX.Element
  dataTooltipId: string
}

export function RepositoryPageHeader({ repoMetadata, title, dataTooltipId }: RepositoryPageHeaderProps) {
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
            <Link to={routes.toCODERepository({ repoPath: repoMetadata.path as string })}>{repoMetadata.uid}</Link>
          </Layout.Horizontal>
          <Container padding={{ top: 'xsmall', bottom: 'small' }}>
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
