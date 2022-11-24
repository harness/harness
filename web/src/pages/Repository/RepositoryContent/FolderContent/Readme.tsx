import React from 'react'
import { Container, Color, Layout, Button, FlexExpander, ButtonVariation, Heading } from '@harness/uicore'
import { useHistory } from 'react-router-dom'
import { useGet } from 'restful-react'
import { MarkdownViewer } from 'components/SourceCodeViewer/SourceCodeViewer'
import { useAppContext } from 'AppContext'
import type { OpenapiContentInfo, OpenapiGetContentOutput, RepoFileContent, TypesRepository } from 'services/code'
import { CodeIcon } from 'utils/GitUtils'
import css from './Readme.module.scss'

interface FolderContentProps {
  metadata: TypesRepository
  gitRef?: string
  readmeInfo: OpenapiContentInfo
}

export function Readme({ metadata, gitRef, readmeInfo }: FolderContentProps) {
  const history = useHistory()
  const { routes } = useAppContext()

  const { data /*error, loading, refetch, response */ } = useGet<OpenapiGetContentOutput>({
    path: `/api/v1/repos/${metadata.path}/+/content/${readmeInfo.path}`,
    queryParams: {
      include_commit: false,
      git_ref: gitRef
    }
  })

  return (
    <Container className={css.readmeContainer} background={Color.WHITE}>
      <Layout.Horizontal padding="small" className={css.heading}>
        <Heading level={5}>{readmeInfo.name}</Heading>
        <FlexExpander />
        <Button
          variation={ButtonVariation.ICON}
          icon={CodeIcon.Edit}
          onClick={() => {
            history.push(
              routes.toCODERepositoryFileEdit({
                repoPath: metadata.path as string,
                gitRef: gitRef || (metadata.defaultBranch as string),
                resourcePath: readmeInfo.path as string
              })
            )
          }}
        />
      </Layout.Horizontal>

      {/* TODO: Loading and Error handling */}
      {(data?.content as RepoFileContent)?.data && (
        <Container className={css.readmeContent}>
          <MarkdownViewer source={window.atob((data?.content as RepoFileContent)?.data || '')} />
        </Container>
      )}
    </Container>
  )
}
