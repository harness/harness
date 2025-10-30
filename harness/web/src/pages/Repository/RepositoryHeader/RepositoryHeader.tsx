/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react'
import { Render } from 'react-jsx-match'
import { Layout, Text } from '@harnessio/uicore'
import { BookmarkBook } from 'iconoir-react'
import { FontVariation } from '@harnessio/design-system'
import { RepoTypeLabel } from 'components/RepoTypeLabel/RepoTypeLabel'
import { useAppContext } from 'AppContext'
import type { GitInfoProps } from 'utils/GitUtils'
import FavoriteStar from 'components/FavoriteStar/FavoriteStar'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import type { RepoRepositoryOutput } from 'services/code'
import css from './RepositoryHeader.module.scss'

interface RepositoryHeaderProps extends Pick<GitInfoProps, 'repoMetadata'> {
  repoMetadata: RepoRepositoryOutput
  isFile: boolean
  updateRepoMetadata: (repoPath: string, field: keyof RepoRepositoryOutput, value: any) => void
  className?: string
}

export function RepositoryHeader(props: RepositoryHeaderProps) {
  const { isCurrentSessionPublic } = useAppContext()
  const { repoMetadata, className, isFile, updateRepoMetadata } = props
  return (
    <RepositoryPageHeader
      className={isFile ? className : undefined}
      repoMetadata={repoMetadata}
      title={
        <Layout.Horizontal spacing="small" className={css.name}>
          <span className={css.customIcon}>
            <BookmarkBook />
          </span>
          <Text inline className={css.repoDropdown} font={{ variation: FontVariation.H4 }}>
            {repoMetadata.identifier}
          </Text>
          <Render when={!isCurrentSessionPublic}>
            <FavoriteStar
              isFavorite={repoMetadata.is_favorite}
              resourceId={repoMetadata.id || 0}
              resourceType={'REPOSITORY'}
              key={repoMetadata.id}
              onChange={favorite => updateRepoMetadata(repoMetadata.path || '', 'is_favorite', favorite)}
            />
          </Render>
          <RepoTypeLabel isPublic={repoMetadata.is_public} isArchived={repoMetadata.archived} />
        </Layout.Horizontal>
      }
      dataTooltipId="repositoryTitle"
    />
  )
}
