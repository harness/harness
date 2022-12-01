import React from 'react'
import { Container, Layout, Text, Color, Icon, FontVariation } from '@harness/uicore'
import { Link } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useAppContext } from 'AppContext'
import type { GitInfoProps } from 'utils/GitUtils'

import css from './RepositoryCreateWebhook.module.scss'

export function RepositoryCreateWebhookHeader({ repoMetadata }: Pick<GitInfoProps, 'repoMetadata'>) {
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
          <Icon name="main-chevron-right" size={10} color={Color.GREY_500} />
          <Link to={routes.toCODESettings({ repoPath: repoMetadata.path as string })}>Settings</Link>
        </Layout.Horizontal>
        <Container padding={{ top: 'medium', bottom: 'medium' }}>
          <Text font={{ variation: FontVariation.H4 }}>
            <Icon name="main-chevron-right" size={10} color={Color.GREY_500} className={css.addVerticalAlign} />
            {getString('webhook')}
          </Text>
        </Container>
      </Container>
    </Container>
  )
}
