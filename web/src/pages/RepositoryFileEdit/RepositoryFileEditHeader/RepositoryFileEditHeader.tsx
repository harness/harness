import React from 'react'
import { Container, Layout, Text, Color, Icon, FontVariation } from '@harness/uicore'
import { Link } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { GitInfoProps, isFile } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
import css from './RepositoryFileEditHeader.module.scss'

export function RepositoryFileEditHeader({
  repoMetadata,
  resourceContent
}: Pick<GitInfoProps, 'repoMetadata' | 'resourceContent'>) {
  const { getString } = useStrings()
  const space = useGetSpaceParam()
  const { routes } = useAppContext()

  return (
    <Container className={css.header}>
      <Container>
        <Layout.Horizontal spacing="small" className={css.breadcrumb}>
          <Link to={routes.toCODERepositories({ space })}>{getString('repositories')}</Link>
          <Icon name="main-chevron-right" size={10} color={Color.GREY_500} />
          <Link to={routes.toCODERepository({ repoPath: repoMetadata.path as string })}>{repoMetadata.uid}</Link>
        </Layout.Horizontal>
        <Container padding={{ top: 'medium', bottom: 'medium' }}>
          <Text font={{ variation: FontVariation.H4 }}>
            {getString(isFile(resourceContent) ? 'editFile' : 'newFile')}
          </Text>
        </Container>
      </Container>
    </Container>
  )
}
