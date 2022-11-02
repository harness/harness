import React from 'react'
import { Container, Layout, Text, Color, Icon, FontVariation } from '@harness/uicore'
import { Link, useParams } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import type { TypesRepository } from 'services/scm'
import { useAppContext } from 'AppContext'
import type { SCMPathProps } from 'RouteDefinitions'
import css from './RepositoryCommitsHeader.module.scss'

interface RepositoryCommitsHeaderProps {
  repoMetadata: TypesRepository
}

export function RepositoryCommitsHeader({ repoMetadata }: RepositoryCommitsHeaderProps): JSX.Element {
  const { getString } = useStrings()
  const { space: spaceFromPath = '' } = useParams<SCMPathProps>()
  const { space = spaceFromPath || '', routes } = useAppContext()

  return (
    <Container className={css.header}>
      <Container>
        <Layout.Horizontal spacing="small" className={css.breadcrumb}>
          <Link to={routes.toSCMRepositoriesListing({ space })}>{getString('repositories')}</Link>
          <Icon name="main-chevron-right" size={10} color={Color.GREY_500} />
          <Link to={routes.toSCMRepository({ repoPath: repoMetadata.path as string })}>{repoMetadata.name}</Link>
        </Layout.Horizontal>
        <Container padding={{ top: 'medium', bottom: 'medium' }}>
          <Text font={{ variation: FontVariation.H4 }}>{getString('commits')}</Text>
        </Container>
      </Container>
    </Container>
  )
}
