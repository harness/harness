import React from 'react'
import { Button, ButtonVariation, Color, Container, FlexExpander, Heading, Layout } from '@harness/uicore'
import { useHistory } from 'react-router-dom'
import { SourceCodeViewer } from 'components/SourceCodeViewer/SourceCodeViewer'
import type { OpenapiGetContentOutput, RepoFileContent, TypesRepository } from 'services/scm'
import { GitIcon } from 'utils/GitUtils'
import { filenameToLanguage } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { LatestCommit } from 'components/LatestCommit/LatestCommit'
import css from './FileContent.module.scss'

interface FileContentProps {
  repoMetadata: TypesRepository
  gitRef: string
  resourcePath: string
  contentInfo: OpenapiGetContentOutput
}

export function FileContent({ repoMetadata, gitRef, resourcePath, contentInfo }: FileContentProps) {
  const { routes } = useAppContext()
  const history = useHistory()
  return (
    <Layout.Vertical spacing="small">
      <LatestCommit repoMetadata={repoMetadata} latestCommit={contentInfo.latestCommit} standaloneStyle />
      <Container className={css.container} background={Color.WHITE}>
        <Layout.Horizontal padding="small" className={css.heading}>
          <Heading level={5}>{contentInfo.name}</Heading>
          <FlexExpander />
          <Button
            variation={ButtonVariation.ICON}
            icon={GitIcon.EDIT}
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
        </Layout.Horizontal>

        {(contentInfo?.content as RepoFileContent)?.data && (
          <Container className={css.content}>
            <SourceCodeViewer
              language={filenameToLanguage(contentInfo?.name)}
              source={window.atob((contentInfo?.content as RepoFileContent)?.data || '')}
            />
          </Container>
        )}
      </Container>
    </Layout.Vertical>
  )
}
