import React from 'react'
import { Button, ButtonVariation, Color, Container, FlexExpander, Heading, Layout } from '@harness/uicore'
import { SourceCodeViewer } from 'components/SourceCodeViewer/SourceCodeViewer'
import type { OpenapiGetContentOutput, RepoFileContent } from 'services/scm'
import { GitIcon } from 'utils/GitUtils'
import { filenameToLanguage } from 'utils/Utils'
import { LatestCommit } from '../LatestCommit/LatestCommit'
import css from './FileContent.module.scss'

interface FileContentProps {
  contentInfo: OpenapiGetContentOutput
}

export function FileContent({ contentInfo }: FileContentProps): JSX.Element {
  return (
    <Layout.Vertical spacing="small">
      <LatestCommit latestCommit={contentInfo.latestCommit} standaloneStyle />
      <Container className={css.container} background={Color.WHITE}>
        <Layout.Horizontal padding="small" className={css.heading}>
          <Heading level={5}>{contentInfo.name}</Heading>
          <FlexExpander />
          <Button variation={ButtonVariation.ICON} icon={GitIcon.EDIT} />
        </Layout.Horizontal>

        {/* TODO: Loading and Error handling */}
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
