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
import { Container, Layout, Text } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { Link } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { GitInfoProps, isFile } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
import { useDocumentTitle } from 'hooks/useDocumentTitle'
import css from './RepositoryFileEditHeader.module.scss'

interface RepositoryFileEditHeaderProps extends Pick<GitInfoProps, 'repoMetadata'> {
  resourceContent: GitInfoProps['resourceContent'] | null
}

export const RepositoryFileEditHeader: React.FC<RepositoryFileEditHeaderProps> = ({
  repoMetadata,
  resourceContent
}) => {
  const { getString } = useStrings()
  const space = useGetSpaceParam()
  const { routes } = useAppContext()
  useDocumentTitle(
    getString(isFile(resourceContent) ? 'pageTitle.editFileLocation' : 'newFile', { path: resourceContent?.path })
  )

  return (
    <Container className={css.header}>
      <Container>
        <Layout.Horizontal spacing="small" className={css.breadcrumb}>
          <Link to={routes.toCODERepositories({ space })}>{getString('repositories')}</Link>
          <Icon name="main-chevron-right" size={10} color={Color.GREY_500} />
          <Link to={routes.toCODERepository({ repoPath: repoMetadata.path as string })}>{repoMetadata.identifier}</Link>
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
