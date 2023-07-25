import React, { useMemo, useState } from 'react'
import { Container, Layout } from '@harness/uicore'
import { Render } from 'react-jsx-match'
import { useHistory, useRouteMatch } from 'react-router-dom'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { routes } from 'RouteDefinitions'
import type { TypesSpace } from 'services/code'
import { SpaceSelector } from 'components/SpaceSelector/SpaceSelector'
import { NavMenuItem } from './NavMenuItem'
import css from './DefaultMenu.module.scss'

export const DefaultMenu: React.FC = () => {
  const history = useHistory()
  const [selectedSpace, setSelectedSpace] = useState<TypesSpace | undefined>()
  const { repoMetadata, gitRef, commitRef } = useGetRepositoryMetadata()
  const { getString } = useStrings()
  const repoPath = useMemo(() => repoMetadata?.path || '', [repoMetadata])
  const routeMatch = useRouteMatch()
  const isFilesSelected = useMemo(
    () => routeMatch.path === '/:space*/:repoName' || routeMatch.path.startsWith('/:space*/:repoName/edit'),
    [routeMatch]
  )

  return (
    <Container className={css.main}>
      <Layout.Vertical spacing="small">
        <SpaceSelector
          onSelect={(_selectedSpace, isUserAction) => {
            setSelectedSpace(_selectedSpace)

            if (isUserAction) {
              history.push(routes.toCODERepositories({ space: _selectedSpace.path as string }))
            }
          }}
        />

        <Render when={selectedSpace}>
          <NavMenuItem
            icon="code-repo"
            rightIcon={repoMetadata ? 'main-chevron-down' : 'main-chevron-right'}
            textProps={{
              rightIconProps: {
                size: 10,
                style: {
                  flexGrow: 1,
                  justifyContent: 'end',
                  display: 'flex'
                }
              }
            }}
            label={getString('repositories')}
            to={routes.toCODERepositories({ space: selectedSpace?.path as string })}
            isDeselected={!!repoMetadata}
            isHighlighted={!!repoMetadata}
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
                to={routes.toCODERepository({ repoPath, gitRef: commitRef || gitRef })}
              />

              <NavMenuItem
                data-code-repo-section="commits"
                isSubLink
                label={getString('commits')}
                to={routes.toCODECommits({
                  repoPath,
                  commitRef: commitRef || gitRef
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
                label={getString('webhooks')}
                to={routes.toCODEWebhooks({
                  repoPath
                })}
              />

              <NavMenuItem
                data-code-repo-section="settings"
                isSubLink
                label={getString('settings')}
                to={routes.toCODESettings({
                  repoPath
                })}
              />
            </Layout.Vertical>
          </Container>
        </Render>

        <Render when={selectedSpace}>
          <NavMenuItem
            icon="nav-project"
            label={getString('accessControl')}
            to={routes.toCODESpaceAccessControl({ space: selectedSpace?.path as string })}
          />

          <NavMenuItem
            icon="code-settings"
            label={getString('settings')}
            to={routes.toCODESpaceSettings({ space: selectedSpace?.path as string })}
          />
        </Render>
      </Layout.Vertical>
    </Container>
  )
}
