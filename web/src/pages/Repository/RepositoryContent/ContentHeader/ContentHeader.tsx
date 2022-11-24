import React, { useEffect, useState } from 'react'
import { Container, Layout, Button, FlexExpander, ButtonVariation, Text, Icon, Color } from '@harness/uicore'
import ReactJoin from 'react-join'
import { Link, useHistory } from 'react-router-dom'
import { useGet } from 'restful-react'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { CloneButtonTooltip } from 'components/CloneButtonTooltip/CloneButtonTooltip'
import { CodeIcon, GitInfoProps, GitRefType, isDir } from 'utils/GitUtils'
import { BranchTagSelect } from 'components/BranchTagSelect/BranchTagSelect'
import { useCreateBranchModal } from 'components/CreateBranchModal/CreateBranchModal'
import css from './ContentHeader.module.scss'

export function ContentHeader({
  repoMetadata,
  gitRef = repoMetadata.defaultBranch as string,
  resourcePath,
  resourceContent
}: Pick<GitInfoProps, 'repoMetadata' | 'gitRef' | 'resourcePath' | 'resourceContent'>) {
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const history = useHistory()
  const [gitRefType, setGitRefType] = useState(GitRefType.BRANCH)
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

  useVerifyGitRefATag({ repoMetadata, gitRef, setGitRefType })

  return (
    <Container className={css.main}>
      <Layout.Horizontal spacing="medium">
        <BranchTagSelect
          repoMetadata={repoMetadata}
          gitRef={gitRef}
          gitRefType={gitRefType}
          onSelect={(ref, type) => {
            setGitRefType(type)
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
              tooltip={<CloneButtonTooltip httpsURL={repoMetadata.gitUrl as string} />}
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
              onClick={() => {
                history.push(
                  routes.toCODERepositoryFileEdit({
                    repoPath: repoMetadata.path as string,
                    resourcePath,
                    gitRef: gitRef || (repoMetadata.defaultBranch as string)
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

interface UseVerifyGitRefATagProps extends Pick<GitInfoProps, 'repoMetadata' | 'gitRef'> {
  setGitRefType: React.Dispatch<React.SetStateAction<GitRefType>>
}

// Since API does not have any field to determine if a content belongs to a branch or a tag. We need
// to do a query to check if a gitRef is a tag by sending an optional API call to /tags and verify
// if the exact tag is returned.
function useVerifyGitRefATag({ repoMetadata, gitRef, setGitRefType }: UseVerifyGitRefATagProps) {
  const { data, refetch } = useGet<{ name: string }[]>({
    path: `/api/v1/repos/${repoMetadata.path}/+/tags`,
    queryParams: {
      per_page: 1,
      page: 1,
      include_commit: false,
      query: gitRef
    },
    lazy: true
  })

  useEffect(() => {
    if (gitRef) {
      refetch()
    }
  }, [gitRef, refetch])

  useEffect(() => {
    if (data?.[0]?.name === gitRef) {
      setGitRefType(GitRefType.TAG)
    } else {
      setGitRefType(GitRefType.BRANCH)
    }
  }, [gitRef, setGitRefType, data])
}
