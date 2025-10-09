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

import React, { useRef } from 'react'
import { Container, PageBody } from '@harnessio/uicore'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useGetResourceContent } from 'hooks/useGetResourceContent'
import { useDisableCodeMainLinks } from 'hooks/useDisableCodeMainLinks'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { voidFn, getErrorMessage } from 'utils/Utils'
import { normalizeGitRef } from 'utils/GitUtils'
import { useSetPageContainerWidthVar } from 'hooks/useSetPageContainerWidthVar'
import { RepositoryFileEditHeader } from './RepositoryFileEditHeader/RepositoryFileEditHeader'
import { FileEditor } from './FileEditor/FileEditor'
import css from './RepositoryFileEdit.module.scss'

export default function RepositoryFileEdit() {
  const { gitRef, resourcePath, repoMetadata, error, loading, refetch } = useGetRepositoryMetadata()
  const {
    data: resourceContent,
    error: resourceError,
    loading: resourceLoading,
    isRepositoryEmpty
  } = useGetResourceContent({ repoMetadata, gitRef: normalizeGitRef(gitRef) as string, resourcePath })

  useDisableCodeMainLinks(!!isRepositoryEmpty)

  const domRef = useRef<HTMLDivElement>(null)
  useSetPageContainerWidthVar({ domRef })

  return (
    <Container className={css.main} ref={domRef}>
      <PageBody error={getErrorMessage(error || resourceError)} retryOnError={voidFn(refetch)}>
        <LoadingSpinner visible={loading && resourceLoading} withBorder={!!resourceContent && resourceLoading} />

        {repoMetadata ? (
          <>
            <RepositoryFileEditHeader repoMetadata={repoMetadata} resourceContent={resourceContent} />
            <Container className={css.resourceContent}>
              {(resourceContent || isRepositoryEmpty) && (
                <FileEditor
                  repoMetadata={repoMetadata}
                  gitRef={gitRef}
                  resourcePath={resourcePath}
                  resourceContent={resourceContent}
                  isRepositoryEmpty={isRepositoryEmpty}
                />
              )}
            </Container>
          </>
        ) : null}
      </PageBody>
    </Container>
  )
}
