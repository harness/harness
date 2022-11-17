import React, { useMemo } from 'react'
import { Button, ButtonVariation, Color, Container, FlexExpander, Heading, Layout, Utils } from '@harness/uicore'
import { useHistory } from 'react-router-dom'
import { SourceCodeViewer } from 'components/SourceCodeViewer/SourceCodeViewer'
import type { RepoFileContent } from 'services/scm'
import { GitIcon, GitInfoProps } from 'utils/GitUtils'
import { filenameToLanguage } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { LatestCommitForFile } from 'components/LatestCommit/LatestCommit'
import { CommitModalButton } from 'components/CommitModalButton/CommitModalButton'
import { useStrings } from 'framework/strings'
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

  return (
    <Layout.Vertical spacing="small">
      <LatestCommitForFile repoMetadata={repoMetadata} latestCommit={resourceContent.latestCommit} standaloneStyle />
      <Container className={css.container} background={Color.WHITE}>
        <Layout.Horizontal padding="small" className={css.heading}>
          <Heading level={5} color={Color.BLACK}>
            {resourceContent.name}
          </Heading>
          <FlexExpander />
          <Layout.Horizontal spacing="xsmall">
            <Button
              variation={ButtonVariation.ICON}
              icon={GitIcon.CodeEdit}
              tooltip={getString('edit')}
              tooltipProps={{ isDark: true }}
              onClick={() => {
                history.push(
                  routes.toSCMRepositoryFileEdit({
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
              icon={GitIcon.CodeCopy}
              tooltipProps={{ isDark: true }}
              onClick={() => Utils.copy(content)}
            />
            <CommitModalButton
              variation={ButtonVariation.ICON}
              icon={GitIcon.CodeDelete}
              tooltipProps={{ isDark: true }}
              tooltip={getString('delete')}
              commitMessagePlaceHolder={getString('deleteFile').replace('__filePath__', resourcePath)}
              gitRef={gitRef}
              resourcePath={resourcePath}
              onSubmit={data => console.log({ data })}
              deleteFile
            />
          </Layout.Horizontal>
        </Layout.Horizontal>

        {(resourceContent?.content as RepoFileContent)?.data && (
          <Container className={css.content}>
            <SourceCodeViewer language={filenameToLanguage(resourceContent?.name)} source={content} />
          </Container>
        )}
      </Container>
    </Layout.Vertical>
  )
}
