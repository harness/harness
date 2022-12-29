import React from 'react'
import { Container, Layout, Button, FlexExpander, ButtonVariation, Text, Icon, Color } from '@harness/uicore'
import ReactJoin from 'react-join'
import { Link, useHistory } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { CloneButtonTooltip } from 'components/CloneButtonTooltip/CloneButtonTooltip'
import { CodeIcon, GitInfoProps, isDir, isRefATag } from 'utils/GitUtils'
import { BranchTagSelect } from 'components/BranchTagSelect/BranchTagSelect'
import { useCreateBranchModal } from 'components/CreateBranchModal/CreateBranchModal'
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
        <Container>
          <Layout.Horizontal spacing="small">
            <Link to={routes.toCODERepository({ repoPath: repoMetadata.path as string, gitRef })}>
              <Icon name={CodeIcon.Folder} />
            </Link>
            <Text color={Color.GREY_900}>/</Text>
            <ReactJoin separator={<Text color={Color.GREY_900}>/</Text>}>
              {resourcePath.split('/').map((_path, index, paths) => {
                const pathAtIndex = paths.slice(0, index + 1).join('/')

                return (
                  <Link
                    key={_path + index}
                    to={routes.toCODERepository({
                      repoPath: repoMetadata.path as string,
                      gitRef,
                      resourcePath: pathAtIndex
                    })}>
                    <Text color={Color.GREY_900}>{_path}</Text>
                  </Link>
                )
              })}
            </ReactJoin>
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
            />
          </>
        )}
      </Layout.Horizontal>
    </Container>
  )
}
