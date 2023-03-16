import React from 'react'
import { Container, Color, Layout, Button, FlexExpander, ButtonVariation, Heading, Icon } from '@harness/uicore'
import { Render } from 'react-jsx-match'
import { useHistory } from 'react-router-dom'
import { useGet } from 'restful-react'
import cx from 'classnames'
import { MarkdownViewer } from 'components/MarkdownViewer/MarkdownViewer'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import type { OpenapiContentInfo, OpenapiGetContentOutput, RepoFileContent, TypesRepository } from 'services/code'
import { useShowRequestError } from 'hooks/useShowRequestError'
import { CodeIcon, decodeGitContent } from 'utils/GitUtils'
import css from './Readme.module.scss'

interface FolderContentProps {
  metadata: TypesRepository
  gitRef?: string
  readmeInfo: OpenapiContentInfo
  contentOnly?: boolean
  maxWidth?: string
}

function ReadmeViewer({ metadata, gitRef, readmeInfo, contentOnly, maxWidth }: FolderContentProps) {
  const history = useHistory()
  const { getString } = useStrings()
  const { routes } = useAppContext()

  const { data, error, loading } = useGet<OpenapiGetContentOutput>({
    path: `/api/v1/repos/${metadata.path}/+/content/${readmeInfo?.path}`,
    queryParams: {
      include_commit: false,
      git_ref: gitRef
    }
  })

  useShowRequestError(error)

  return (
    <Container
      className={cx(css.readmeContainer, { [css.contentOnly]: contentOnly })}
      background={Color.WHITE}
      style={{ '--max-width': maxWidth } as React.CSSProperties}>
      <Render when={!contentOnly}>
        <Layout.Horizontal padding="small" className={css.heading}>
          <Heading level={5}>{readmeInfo.name}</Heading>
          <FlexExpander />
          {loading && <Icon name="spinner" color={Color.PRIMARY_7} />}
          <Button
            variation={ButtonVariation.ICON}
            icon={CodeIcon.Edit}
            onClick={() => {
              history.push(
                routes.toCODEFileEdit({
                  repoPath: metadata.path as string,
                  gitRef: gitRef || (metadata.default_branch as string),
                  resourcePath: readmeInfo.path as string
                })
              )
            }}
          />
        </Layout.Horizontal>
      </Render>

      <Render when={(data?.content as RepoFileContent)?.data}>
        <Container className={css.readmeContent}>
          <MarkdownViewer source={decodeGitContent((data?.content as RepoFileContent)?.data)} getString={getString} />
        </Container>
      </Render>
    </Container>
  )
}

export const Readme = React.memo(ReadmeViewer)
