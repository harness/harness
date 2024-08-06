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

import React, { useMemo, useState } from 'react'
import { Container, Layout } from '@harnessio/uicore'
import { Render } from 'react-jsx-match'
import { useHistory, useRouteMatch } from 'react-router-dom'
import { FingerprintLockCircle, BookmarkBook, UserSquare, Settings } from 'iconoir-react'
import { useGet } from 'restful-react'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import type { SpaceSpaceOutput } from 'services/code'
import { SpaceSelector } from 'components/SpaceSelector/SpaceSelector'
import { useAppContext } from 'AppContext'
import { isGitRev } from 'utils/GitUtils'
import { NavMenuItem } from './NavMenuItem'
import css from './DefaultMenu.module.scss'

export const DefaultMenu: React.FC = () => {
  const history = useHistory()
  const { routes, standalone, isCurrentSessionPublic } = useAppContext()
  const [selectedSpace, setSelectedSpace] = useState<SpaceSpaceOutput | undefined>()
  const { repoMetadata, gitRef, commitRef } = useGetRepositoryMetadata()
  const { getString } = useStrings()
  const repoPath = useMemo(() => repoMetadata?.path || '', [repoMetadata])
  const routeMatch = useRouteMatch()
  const isCommitSelected = useMemo(() => routeMatch.path === '/:space*/:repoName/commit/:commitRef*', [routeMatch])

  const { data: systemConfig } = useGet({ path: 'api/v1/system/config' })

  const isFilesSelected = useMemo(
    () =>
      !isCommitSelected &&
      (routeMatch.path === '/:space*/:repoName' || routeMatch.path.startsWith('/:space*/:repoName/edit')),
    [routeMatch, isCommitSelected]
  )
  const isWebhookSelected = useMemo(() => routeMatch.path.startsWith('/:space*/:repoName/webhook'), [routeMatch])
  const _gitRef = useMemo(() => {
    const ref = commitRef || gitRef
    return !isGitRev(ref) ? ref : ''
  }, [commitRef, gitRef])

  const isSemanticSearchEnabled = false

  return (
    <Container className={css.main}>
      <Layout.Vertical spacing="small">
        <Render when={!isCurrentSessionPublic}>
          <SpaceSelector
            onSelect={(_selectedSpace, isUserAction) => {
              setSelectedSpace(_selectedSpace)
              if (_selectedSpace.path === '' && _selectedSpace.id === -1) {
                setSelectedSpace(undefined)
              }
              if (isUserAction) {
                history.push(routes.toCODERepositories({ space: _selectedSpace.path as string }))
              }
            }}
          />
        </Render>

        <Render when={selectedSpace}>
          <NavMenuItem
            label={getString('repositories')}
            to={routes.toCODERepositories({ space: selectedSpace?.path as string })}
            isDeselected={!!repoMetadata}
            isHighlighted={!!repoMetadata}
            customIcon={<BookmarkBook />}
          />
        </Render>

        <Render when={repoMetadata}>
          <Container className={css.repoLinks}>
            <Layout.Vertical spacing="small">
              <NavMenuItem
                data-code-repo-section="files"
                isSubLink
                isSelected={isFilesSelected}
                label={getString('files')}
                to={routes.toCODERepository({ repoPath, gitRef: _gitRef || repoMetadata?.default_branch })}
              />

              <NavMenuItem
                data-code-repo-section="commits"
                isSelected={isCommitSelected}
                isSubLink
                label={getString('commits')}
                to={routes.toCODECommits({
                  repoPath,
                  commitRef: _gitRef
                })}
              />

              <NavMenuItem
                data-code-repo-section="branches"
                isSubLink
                label={getString('branches')}
                to={routes.toCODEBranches({
                  repoPath
                })}
              />

              <NavMenuItem
                data-code-repo-section="tags"
                isSubLink
                label={getString('tags')}
                to={routes.toCODETags({
                  repoPath
                })}
              />

              <NavMenuItem
                data-code-repo-section="pull-requests"
                isSubLink
                label={getString('pullRequests')}
                to={routes.toCODEPullRequests({
                  repoPath
                })}
              />

              <NavMenuItem
                data-code-repo-section="branches"
                isSubLink
                isSelected={isWebhookSelected}
                label={getString('webhooks')}
                to={routes.toCODEWebhooks({
                  repoPath
                })}
              />

              {standalone && (
                <NavMenuItem
                  data-code-repo-section="pipelines"
                  isSubLink
                  label={getString('pageTitle.pipelines')}
                  to={routes.toCODEPipelines({
                    repoPath
                  })}
                />
              )}

              <NavMenuItem
                data-code-repo-section="settings"
                isSubLink
                label={getString('settings')}
                to={routes.toCODESettings({
                  repoPath
                })}
              />

              {!standalone && (
                <NavMenuItem
                  data-code-repo-section="search"
                  isSubLink
                  label={getString('search')}
                  to={
                    isSemanticSearchEnabled
                      ? routes.toCODESemanticSearch({ repoPath })
                      : `${routes.toCODERepositorySearch({ repoPath })}?q=repo:${repoPath}`
                  }
                />
              )}
            </Layout.Vertical>
          </Container>
        </Render>

        {systemConfig?.gitspace_enabled && (
          <Render when={selectedSpace}>
            <NavMenuItem
              className=""
              label={getString('cde.gitspaces')}
              to={routes.toCDEGitspaces({ space: selectedSpace?.path as string })}
              icon="gitspace"
            />
          </Render>
        )}

        <Render when={!standalone && selectedSpace}>
          <NavMenuItem
            icon="thinner-search"
            data-code-repo-section="search"
            label={getString('search')}
            to={routes.toCODESpaceSearch({ space: selectedSpace?.path as string })}
          />
        </Render>

        {standalone && (
          <Render when={selectedSpace}>
            <NavMenuItem
              label={getString('pageTitle.secrets')}
              to={routes.toCODESecrets({ space: selectedSpace?.path as string })}
              customIcon={<FingerprintLockCircle />}
            />
          </Render>
        )}

        <Render when={selectedSpace}>
          <NavMenuItem
            customIcon={<UserSquare />}
            label={getString('permissions')}
            to={routes.toCODESpaceAccessControl({ space: selectedSpace?.path as string })}
          />

          <NavMenuItem
            customIcon={<Settings />}
            label={getString('settings')}
            to={routes.toCODESpaceSettings({ space: selectedSpace?.path as string })}
          />
        </Render>
      </Layout.Vertical>
    </Container>
  )
}
