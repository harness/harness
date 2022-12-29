import React, { useMemo } from 'react'
import { Button, ButtonVariation, Color, Container, FlexExpander, Heading, Layout, Utils } from '@harness/uicore'
import { useHistory } from 'react-router-dom'
import { SourceCodeViewer } from 'components/SourceCodeViewer/SourceCodeViewer'
import type { RepoFileContent } from 'services/code'
import { CodeIcon, findMarkdownInfo, GitCommitAction, GitInfoProps, isRefATag } from 'utils/GitUtils'
import { filenameToLanguage } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { LatestCommitForFile } from 'components/LatestCommit/LatestCommit'
import { CommitModalButton } from 'components/CommitModalButton/CommitModalButton'
import { useStrings } from 'framework/strings'
import { Readme } from '../FolderContent/Readme'
import css from './FileContent.module.scss'

export function FileContent({
  repoMetadata,
  gitRef,
  resourcePath,
  resourceContent
}: Pick<GitInfoProps, 'repoMetadata' | 'gitRef' | 'resourcePath' | 'resourceContent'>) {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const history = useHistory()
  const content = useMemo(
    () => window.atob((resourceContent?.content as RepoFileContent)?.data || ''),
    [resourceContent?.content]
  )
  const markdownInfo = useMemo(() => findMarkdownInfo(resourceContent), [resourceContent])

  return (
    <Layout.Vertical spacing="small">
      <LatestCommitForFile repoMetadata={repoMetadata} latestCommit={resourceContent.latest_commit} standaloneStyle />
      <Container className={css.container} background={Color.WHITE}>
        <Layout.Horizontal padding="small" className={css.heading}>
          <Heading level={5} color={Color.BLACK}>
            {resourceContent.name}
          </Heading>
          <FlexExpander />
          <Layout.Horizontal spacing="xsmall">
            <Button
              variation={ButtonVariation.ICON}
              icon={CodeIcon.Edit}
              tooltip={isRefATag(gitRef) ? getString('editNotAllowed') : getString('edit')}
              tooltipProps={{ isDark: true }}
              disabled={isRefATag(gitRef)}
              onClick={() => {
                history.push(
                  routes.toCODEFileEdit({
                    repoPath: repoMetadata.path as string,
                    gitRef,
                    resourcePath
                  })
                )
              }}
            />
            <Button
              variation={ButtonVariation.ICON}
              tooltip={getString('copy')}
              icon={CodeIcon.Copy}
              tooltipProps={{ isDark: true }}
              onClick={() => Utils.copy(content)}
            />
            <CommitModalButton
              variation={ButtonVariation.ICON}
              icon={CodeIcon.Delete}
              disabled={isRefATag(gitRef)}
              tooltip={getString(isRefATag(gitRef) ? 'deleteNotAllowed' : 'delete')}
              tooltipProps={{ isDark: true }}
              repoMetadata={repoMetadata}
              gitRef={gitRef}
              resourcePath={resourcePath}
              commitAction={GitCommitAction.DELETE}
              commitTitlePlaceHolder={getString('deleteFile').replace('__path__', resourcePath)}
              onSuccess={(_commitInfo, newBranch) => {
                history.push(
                  routes.toCODERepository({
                    repoPath: repoMetadata.path as string,
                    gitRef: newBranch || gitRef
                  })
                )
              }}
            />
          </Layout.Horizontal>
        </Layout.Horizontal>

        {(resourceContent?.content as RepoFileContent)?.data && (
          <Container className={css.content}>
            {!markdownInfo ? (
              <SourceCodeViewer language={filenameToLanguage(resourceContent?.name)} source={content} />
            ) : (
              <Readme metadata={repoMetadata} readmeInfo={markdownInfo} contentOnly maxWidth="calc(100vw - 346px)" />
            )}
          </Container>
        )}
      </Container>
    </Layout.Vertical>
  )
}
