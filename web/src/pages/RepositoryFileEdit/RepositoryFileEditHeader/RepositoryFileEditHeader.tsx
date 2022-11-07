import React from 'react'
import { Container, Layout, Text, Color, Icon, FontVariation } from '@harness/uicore'
import { Link } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import type { TypesRepository } from 'services/scm'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useAppContext } from 'AppContext'
import css from './RepositoryFileEditHeader.module.scss'

interface RepositoryFileEditHeaderProps {
  repoMetadata: TypesRepository
  resourcePath: string
}

export function RepositoryFileEditHeader({ repoMetadata, resourcePath }: RepositoryFileEditHeaderProps) {
  const { getString } = useStrings()
  const space = useGetSpaceParam()
  const { routes } = useAppContext()

  return (
    <Container className={css.header}>
      <Container>
        <Layout.Horizontal spacing="small" className={css.breadcrumb}>
          <Link to={routes.toSCMRepositoriesListing({ space })}>{getString('repositories')}</Link>
          <Icon name="main-chevron-right" size={10} color={Color.GREY_500} />
          <Link to={routes.toSCMRepository({ repoPath: repoMetadata.path as string })}>{repoMetadata.name}</Link>
        </Layout.Horizontal>
        <Container padding={{ top: 'medium', bottom: 'medium' }}>
          <Text font={{ variation: FontVariation.H4 }}>{getString(resourcePath ? 'editFile' : 'newFile')}</Text>
        </Container>
      </Container>
    </Container>
  )
}
