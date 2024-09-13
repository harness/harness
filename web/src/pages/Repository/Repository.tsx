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

import React, { useEffect, useRef, useState } from 'react'
import { Button, ButtonVariation, Container, Layout, PageBody, StringSubstitute, Text } from '@harnessio/uicore'
import { Falsy, Match, Truthy } from 'react-jsx-match'
import cx from 'classnames'
import { useHistory } from 'react-router-dom'
import { useGetResourceContent } from 'hooks/useGetResourceContent'
import { getErrorMessage } from 'utils/Utils'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { useStrings } from 'framework/strings'
import type { OpenapiGetContentOutput, RepoRepositoryOutput } from 'services/code'
import { Images } from 'images'
import { useSetPageContainerWidthVar } from 'hooks/useSetPageContainerWidthVar'
import { normalizeGitRef, isDir } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
import { RepositoryContent } from './RepositoryContent/RepositoryContent'
import { RepositoryHeader } from './RepositoryHeader/RepositoryHeader'
import { ContentHeader } from './RepositoryContent/ContentHeader/ContentHeader'
import { EmptyRepositoryInfo } from './EmptyRepositoryInfo'
import css from './Repository.module.scss'

enum NotFoundType {
  BRANCH = 'branchNotFound',
  FILE = 'fileNotFound',
  NONE = 'none'
}
const regexPatterns = {
  branchNotFound: /revision "refs\/heads\/[^"]+" not found/,
  pathNotFound: /path '[^']+' wasn't found in the repo/
}

export default function Repository() {
  const { gitRef, resourcePath, repoMetadata, error, loading, refetch, commitRef } = useGetRepositoryMetadata()
  const { routes } = useAppContext()
  const history = useHistory()
  const {
    data: resourceContent,
    error: resourceError,
    loading: resourceLoading,
    isRepositoryEmpty,
    refetch: refetchContent
  } = useGetResourceContent({
    repoMetadata,
    gitRef: normalizeGitRef(gitRef) as string,
    resourcePath,
    includeCommit: true
  })
  const { getString } = useStrings()
  const domRef = useRef<HTMLDivElement>(null)

  const [notFoundError, setNotFoundError] = useState(NotFoundType.NONE)
  const isBranchNotFoundError = () => regexPatterns.branchNotFound.test(resourceError?.data.message as string)
  const isPathNotFoundError = () => regexPatterns.pathNotFound.test(resourceError?.data.message as string)

  useEffect(() => {
    if (resourceError?.status === 404 && !isRepositoryEmpty) {
      if (isPathNotFoundError()) {
        setNotFoundError(NotFoundType.FILE)
      }
      if (isBranchNotFoundError()) {
        setNotFoundError(NotFoundType.BRANCH)
      }
    } else {
      setNotFoundError(NotFoundType.NONE)
    }
  }, [resourceError, repoMetadata, gitRef])

  useSetPageContainerWidthVar({ domRef })

  return (
    <Container className={cx(css.main, !!resourceContent && css.withFileViewer)} ref={domRef}>
      <Match expr={resourceError?.status === 404}>
        <Truthy>
          <RepositoryHeader isFile={false} repoMetadata={repoMetadata as RepoRepositoryOutput} />
          <Layout.Vertical>
            <Container padding={{ left: 'xlarge' }}>
              <ContentHeader
                repoMetadata={repoMetadata as RepoRepositoryOutput}
                gitRef={gitRef}
                resourcePath={resourcePath}
                resourceContent={resourceContent as OpenapiGetContentOutput}
              />
              {notFoundError === NotFoundType.FILE && (
                <Container className={css.bannerContainer} padding={{ left: 'xlarge' }}>
                  <Text font={'small'} padding={{ left: 'large' }}>
                    <StringSubstitute
                      str={getString('branchDoesNotHaveFile')}
                      vars={{
                        repoName: repoMetadata?.identifier,
                        fileName: resourcePath,
                        branchName: gitRef
                      }}
                    />
                  </Text>
                </Container>
              )}
            </Container>
            <PageBody
              noData={{
                when: () => notFoundError !== NotFoundType.NONE,
                messageTitle:
                  notFoundError === NotFoundType.BRANCH ? getString('branchNotFoundError') : getString('pageNotFound'),
                message:
                  notFoundError === NotFoundType.BRANCH
                    ? getString('branchNotFoundMessage', { gitRef })
                    : getString('error404Text'),
                image: Images.error404,
                button: (
                  <Button
                    variation={ButtonVariation.PRIMARY}
                    text={getString('goToDefaultBranch')}
                    onClick={() => {
                      history.replace(
                        routes.toCODERepository({
                          repoPath: repoMetadata?.path as string,
                          gitRef: repoMetadata?.default_branch
                        })
                      )
                    }}
                  />
                )
              }}></PageBody>
          </Layout.Vertical>
        </Truthy>
        <Falsy>
          <PageBody
            error={getErrorMessage(error || resourceError)}
            retryOnError={() => (error ? refetch() : refetchContent())}>
            <LoadingSpinner
              visible={loading || resourceLoading}
              withBorder={!!resourceContent && resourceLoading}
              className={cx(css.spinner, { [css.withBorder]: !!resourceContent && resourceLoading })}
            />

            {!!repoMetadata && (
              <>
                <RepositoryHeader
                  repoMetadata={repoMetadata}
                  isFile={!isDir(resourceContent)}
                  className={css.headerContainer}
                />
                {!!resourceContent && (
                  <RepositoryContent
                    repoMetadata={repoMetadata}
                    gitRef={gitRef}
                    resourcePath={resourcePath}
                    resourceContent={resourceContent}
                    commitRef={commitRef}
                  />
                )}
                {isRepositoryEmpty && <EmptyRepositoryInfo repoMetadata={repoMetadata} />}
              </>
            )}
          </PageBody>
        </Falsy>
      </Match>
    </Container>
  )
}
