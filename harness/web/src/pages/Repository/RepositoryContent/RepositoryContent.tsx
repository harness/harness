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

import React, { useEffect } from 'react'
import { Container } from '@harnessio/uicore'
import { GitInfoProps, isDir } from 'utils/GitUtils'
import { ContentHeader } from './ContentHeader/ContentHeader'
import { FolderContent } from './FolderContent/FolderContent'
import { FileContent } from './FileContent/FileContent'
import css from './RepositoryContent.module.scss'

export function RepositoryContent({
  repoMetadata,
  gitRef,
  resourcePath,
  resourceContent,
  commitRef
}: Pick<GitInfoProps, 'repoMetadata' | 'gitRef' | 'resourcePath' | 'resourceContent' | 'commitRef'>) {
  useEffect(() => window.scroll({ top: 0 }), [gitRef, resourcePath])

  return (
    <Container className={css.resourceContent}>
      <ContentHeader
        repoMetadata={repoMetadata}
        gitRef={gitRef}
        resourcePath={resourcePath}
        resourceContent={resourceContent}
      />
      {(isDir(resourceContent) && (
        <FolderContent
          resourceContent={resourceContent}
          resourcePath={resourcePath}
          repoMetadata={repoMetadata}
          gitRef={gitRef || (repoMetadata.default_branch as string)}
        />
      )) || (
        <FileContent
          repoMetadata={repoMetadata}
          gitRef={gitRef}
          resourcePath={resourcePath}
          resourceContent={resourceContent}
          commitRef={commitRef}
        />
      )}
    </Container>
  )
}
