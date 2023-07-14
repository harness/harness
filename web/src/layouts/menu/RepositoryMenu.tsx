import React, { useMemo } from 'react'
import { Container, Layout } from '@harness/uicore'
import { Render } from 'react-jsx-match'
import { useRouteMatch } from 'react-router-dom'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { routes } from 'RouteDefinitions'
import { NavMenu } from './NavMenu'

export const RepositoryMenu: React.FC = () => {
  const { space, repoMetadata, gitRef } = useGetRepositoryMetadata()
  const { getString } = useStrings()
  const repoPath = useMemo(() => repoMetadata?.path || '', [repoMetadata])
  const routeMatch = useRouteMatch()
  const isFilesSelected = useMemo(
    () => routeMatch.path === '/:space/:repoName' || routeMatch.path.startsWith('/:space/:repoName/edit'),
    [routeMatch]
  )

  return (
    <Container padding={{ top: 'medium' }}>
      <Layout.Vertical spacing="small">
        <NavMenu
          icon={repoMetadata ? 'main-chevron-left' : undefined}
          textProps={{
            iconProps: {
              size: 12
            }
          }}
          label={getString('repositories')}
          to={routes.toCODERepositories({ space })}
          isDeselected={!!repoMetadata}
        />

        <Render when={repoMetadata}>
          <Container>
            <Layout.Vertical spacing="small">
              <NavMenu
                data-code-repo-section="files"
                icon="code-file-light"
                isSubLink
                isSelected={isFilesSelected}
                textProps={{
                  iconProps: {
                    size: 18
                  }
                }}
                label={getString('files')}
                to={routes.toCODERepository({ repoPath, gitRef })}
              />

              <NavMenu
                data-code-repo-section="commits"
                icon="git-commit"
                isSubLink
                textProps={{
                  iconProps: {
                    size: 16
                  }
                }}
                label={getString('commits')}
                to={routes.toCODECommits({
                  repoPath,
                  commitRef: ''
                })}
              />

              <NavMenu
                data-code-repo-section="branches"
                isSubLink
                icon="git-branch"
                textProps={{
                  iconProps: {
                    size: 14
                  }
                }}
                label={getString('branches')}
                to={routes.toCODEBranches({
                  repoPath
                })}
              />

              <NavMenu
                data-code-repo-section="tags"
                isSubLink
                icon="code-tag"
                textProps={{
                  iconProps: {
                    size: 18
                    // className: css.tagIcon
                  }
                }}
                label={getString('tags')}
                to={routes.toCODETags({
                  repoPath
                })}
              />

              <NavMenu
                data-code-repo-section="pull-requests"
                isSubLink
                icon="git-pull"
                textProps={{
                  iconProps: {
                    size: 14
                  }
                }}
                label={getString('pullRequests')}
                to={routes.toCODEPullRequests({
                  repoPath
                })}
              />

              <NavMenu
                data-code-repo-section="branches"
                isSubLink
                icon="code-webhook"
                textProps={{
                  iconProps: {
                    size: 20
                  }
                }}
                label={getString('webhooks')}
                to={routes.toCODEWebhooks({
                  repoPath
                })}
              />

              <NavMenu
                data-code-repo-section="settings"
                isSubLink
                icon="code-settings"
                textProps={{
                  iconProps: {
                    size: 20
                  }
                }}
                label={getString('settings')}
                to={routes.toCODESettings({
                  repoPath
                })}
              />
            </Layout.Vertical>
          </Container>
        </Render>
      </Layout.Vertical>
    </Container>
  )
}
