import React from 'react'
import { Layout, Text } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { FontVariation } from '@harnessio/design-system'
import { RepoPublicLabel } from 'components/RepoPublicLabel/RepoPublicLabel'
import { CodeIcon, GitInfoProps } from 'utils/GitUtils'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import css from './RepositoryHeader.module.scss'

export function RepositoryHeader({ repoMetadata }: Pick<GitInfoProps, 'repoMetadata'>) {
  return (
    <RepositoryPageHeader
      repoMetadata={repoMetadata}
      title={
        <Layout.Horizontal spacing="small" className={css.name}>
          <Icon name={CodeIcon.Repo} size={20} />
          <Text inline className={css.repoDropdown} font={{ variation: FontVariation.H4 }}>
            {repoMetadata.uid}
          </Text>
          <RepoPublicLabel isPublic={repoMetadata.is_public} />
        </Layout.Horizontal>
      }
      dataTooltipId="repositoryTitle"
    />
  )
}
