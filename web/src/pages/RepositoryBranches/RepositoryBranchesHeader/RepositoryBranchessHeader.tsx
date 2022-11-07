import React from 'react'
import { Container, Layout, Text, Color, Icon, FontVariation } from '@harness/uicore'
import { Link } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import type { TypesRepository } from 'services/scm'
import { useAppContext } from 'AppContext'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import css from './RepositoryBranchesHeader.module.scss'

interface RepositoryCommitsHeaderProps {
  repoMetadata: TypesRepository
}

export function RepositoryBranchesHeader({ repoMetadata }: RepositoryCommitsHeaderProps) {
  const { getString } = useStrings()
  const space = useGetSpaceParam()
  const { routes } = useAppContext()

  return (
    <Container className={css.header}>
      <Container>
        <Layout.Horizontal spacing="small" className={css.breadcrumb}>
          <Link to={routes.toSCMRepositoriesListing({ space })}>{getString('repositories')}</Link>
          <Icon name="main-chevron-right" size={10} color={Color.GREY_500} />
          <Link to={routes.toSCMRepository({ repoPath: repoMetadata.path as string })}>{repoMetadata.uid}</Link>
        </Layout.Horizontal>
        <Container padding={{ top: 'medium', bottom: 'medium' }}>
          <Text font={{ variation: FontVariation.H4 }}>{getString('branches')}</Text>
        </Container>
      </Container>
    </Container>
  )
}
