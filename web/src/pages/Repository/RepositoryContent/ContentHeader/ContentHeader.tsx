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

import React, { useMemo } from 'react'
import { Container, Layout, Button, FlexExpander, ButtonVariation, Text, ButtonSize } from '@harnessio/uicore'
import cx from 'classnames'
import { Icon } from '@harnessio/icons'
import { Color } from '@harnessio/design-system'
import { Breadcrumbs, IBreadcrumbProps } from '@blueprintjs/core'
import { Link, useHistory } from 'react-router-dom'
import { compact, isEmpty } from 'lodash-es'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { CloneButtonTooltip } from 'components/CloneButtonTooltip/CloneButtonTooltip'
import { CodeIcon, GitInfoProps, isDir, isGitRev, isRefATag } from 'utils/GitUtils'
import { BranchTagSelect } from 'components/BranchTagSelect/BranchTagSelect'
import { useCreateBranchModal } from 'components/CreateBranchModal/CreateBranchModal'
// import KeywordSearch from 'components/CodeSearch/KeywordSearch'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { permissionProps } from 'utils/Utils'
import CodeSearch from 'components/CodeSearch/CodeSearch'
import { useDocumentTitle } from 'hooks/useDocumentTitle'
import { CopyButton } from 'components/CopyButton/CopyButton'
import css from './ContentHeader.module.scss'

export function ContentHeader({
  repoMetadata,
  gitRef = repoMetadata.default_branch as string,
  resourcePath,
  resourceContent
}: Pick<GitInfoProps, 'repoMetadata' | 'gitRef' | 'resourcePath' | 'resourceContent'>) {
  const { getString } = useStrings()
  const { routes, standalone, hooks, isCurrentSessionPublic } = useAppContext()
  const history = useHistory()
  const _isDir = isDir(resourceContent)
  const space = useGetSpaceParam()
  const repoPath = compact([repoMetadata.identifier, resourceContent?.path])
  useDocumentTitle(isEmpty(resourceContent?.path) ? getString('pageTitle.repository') : repoPath.join('/'))

  const permPushResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY',
        resourceIdentifier: repoMetadata?.identifier as string
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
    <Container className={cx(css.main, { [css.mainContainer]: !isDir(resourceContent) })}>
      <Layout.Horizontal className={isDir(resourceContent) ? '' : css.mainBorder} spacing="medium">
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
        <Container style={{ maxWidth: 'calc(var(--page-container-width) - 450px)' }}>
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
            {resourcePath && <CopyButton content={resourcePath} icon={CodeIcon.Copy} size={ButtonSize.MEDIUM} />}
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
              tooltip={
                <CloneButtonTooltip
                  httpsURL={repoMetadata.git_url as string}
                  sshURL={repoMetadata.git_ssh_url as string}
                />
              }
              tooltipProps={{
                interactionKind: 'click',
                minimal: true,
                position: 'bottom-right'
              }}
            />
            <Button
              text={getString('newFile')}
              style={{ whiteSpace: 'nowrap' }}
              icon={CodeIcon.Add}
              variation={ButtonVariation.PRIMARY}
              disabled={isRefATag(gitRef) || isGitRev(gitRef)}
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
      <div className={css.searchBoxCtn}>
        {!standalone && !isCurrentSessionPublic ? <CodeSearch repoMetadata={repoMetadata} /> : null}
      </div>
    </Container>
  )
}
