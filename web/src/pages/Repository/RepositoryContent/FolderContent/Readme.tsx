import React from 'react'
import { Container, Color, Layout, Button, FlexExpander, ButtonVariation, Heading } from '@harness/uicore'
import { useHistory } from 'react-router-dom'
import { useGet } from 'restful-react'
import { useStrings } from 'framework/strings'
import { MarkdownViewer } from 'components/SourceCodeViewer/SourceCodeViewer'
import { useAppContext } from 'AppContext'
import type { OpenapiContentInfo, OpenapiGetContentOutput, RepoFileContent } from 'services/scm'
import { GitIcon } from 'utils/GitUtils'
import type { RepositoryDTO } from 'types/SCMTypes'
import css from './Readme.module.scss'

interface FolderContentProps {
  metadata: RepositoryDTO
  gitRef?: string
  readmeInfo: OpenapiContentInfo
}

export function Readme({ metadata, gitRef, readmeInfo }: FolderContentProps): JSX.Element {
  const { getString } = useStrings()
  const history = useHistory()
  const { routes } = useAppContext()

  const { data /*error, loading, refetch, response */ } = useGet<OpenapiGetContentOutput>({
    path: `/api/v1/repos/${metadata.path}/+/content/${readmeInfo.path}?include_commit=false${
      gitRef ? `&git_ref=${gitRef}` : ''
    }`
  })

  return (
    <Container className={css.readmeContainer} background={Color.WHITE}>
      <Layout.Horizontal padding="small" className={css.heading}>
        <Heading level={5}>{readmeInfo.name}</Heading>
        <FlexExpander />
        <Button variation={ButtonVariation.ICON} icon={GitIcon.EDIT} />
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
