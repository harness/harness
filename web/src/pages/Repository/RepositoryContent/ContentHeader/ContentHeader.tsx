import React, { useMemo } from 'react'
import { Container, Layout, Button, FlexExpander, ButtonVariation, Text } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color } from '@harnessio/design-system'
import { Breadcrumbs, IBreadcrumbProps } from '@blueprintjs/core'
import { Link, useHistory } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { CloneButtonTooltip } from 'components/CloneButtonTooltip/CloneButtonTooltip'
import { CodeIcon, GitInfoProps, isDir, isRefATag } from 'utils/GitUtils'
import { BranchTagSelect } from 'components/BranchTagSelect/BranchTagSelect'
import { useCreateBranchModal } from 'components/CreateBranchModal/CreateBranchModal'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { permissionProps } from 'utils/Utils'
import css from './ContentHeader.module.scss'

export function ContentHeader({
  repoMetadata,
  gitRef = repoMetadata.default_branch as string,
  resourcePath,
  resourceContent
}: Pick<GitInfoProps, 'repoMetadata' | 'gitRef' | 'resourcePath' | 'resourceContent'>) {
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const history = useHistory()
  const _isDir = isDir(resourceContent)
  const { standalone } = useAppContext()
  const { hooks } = useAppContext()
  const space = useGetSpaceParam()

  const permPushResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY'
      },
      permissions: ['code_repo_push']
    },
    [space]
  )
  const openCreateNewBranchModal = useCreateBranchModal({
    repoMetadata,
    onSuccess: branchInfo => {
      history.push(
        routes.toCODERepository({
          repoPath: repoMetadata.path as string,
          gitRef: branchInfo.name
        })
      )
    },
    suggestedSourceBranch: gitRef,
    showSuccessMessage: true
  })
  const breadcrumbs = useMemo(() => {
    return resourcePath.split('/').map((_path, index, paths) => {
      const pathAtIndex = paths.slice(0, index + 1).join('/')
      const href = routes.toCODERepository({
        repoPath: repoMetadata.path as string,
        gitRef,
        resourcePath: pathAtIndex
      })

      return { href, text: _path }
    })
  }, [resourcePath, gitRef, repoMetadata.path, routes])

  return (
    <Container className={css.main}>
      <Layout.Horizontal spacing="medium">
        <BranchTagSelect
          repoMetadata={repoMetadata}
          gitRef={gitRef}
          onSelect={ref => {
            history.push(
              routes.toCODERepository({
                repoPath: repoMetadata.path as string,
                gitRef: ref,
                resourcePath
              })
            )
          }}
          onCreateBranch={openCreateNewBranchModal}
        />
        <Container style={{ maxWidth: 'calc(100vw - 750px)' }}>
          <Layout.Horizontal spacing="small">
            <Link
              id="repository-ref-root"
              className={css.refRoot}
              to={routes.toCODERepository({ repoPath: repoMetadata.path as string, gitRef })}>
              <Icon name={CodeIcon.Folder} />
            </Link>
            <Text className={css.rootSlash} color={Color.GREY_900}>
              /
            </Text>
            <Breadcrumbs
              items={breadcrumbs}
              breadcrumbRenderer={({ text, href }: IBreadcrumbProps) => {
                return (
                  <Link to={href as string}>
                    <Text color={Color.GREY_900}>{text}</Text>
                  </Link>
                )
              }}
            />
          </Layout.Horizontal>
        </Container>
        <FlexExpander />
        {_isDir && (
          <>
            <Button
              text={getString('clone')}
              variation={ButtonVariation.SECONDARY}
              icon={CodeIcon.Clone}
              className={css.btnColorFix}
              tooltip={<CloneButtonTooltip httpsURL={repoMetadata.git_url as string} />}
              tooltipProps={{
                interactionKind: 'click',
                minimal: true,
                position: 'bottom-right'
              }}
            />
            <Button
              text={getString('newFile')}
              icon={CodeIcon.Add}
              variation={ButtonVariation.PRIMARY}
              disabled={isRefATag(gitRef)}
              tooltip={isRefATag(gitRef) ? getString('newFileNotAllowed') : undefined}
              tooltipProps={{ isDark: true }}
              onClick={() => {
                history.push(
                  routes.toCODEFileEdit({
                    repoPath: repoMetadata.path as string,
                    resourcePath,
                    gitRef: gitRef || (repoMetadata.default_branch as string)
                  })
                )
              }}
              {...permissionProps(permPushResult, standalone)}
            />
          </>
        )}
      </Layout.Horizontal>
    </Container>
  )
}
