import React, { useEffect, useMemo, useRef } from 'react'
import { Container, Layout, FlexExpander, ButtonVariation, Heading, ButtonSize } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color } from '@harnessio/design-system'
import { Render } from 'react-jsx-match'
import { useHistory } from 'react-router-dom'
import { useGet } from 'restful-react'
import cx from 'classnames'
import { MarkdownViewer } from 'components/MarkdownViewer/MarkdownViewer'
import { useAppContext } from 'AppContext'
import type { OpenapiContentInfo, OpenapiGetContentOutput, RepoFileContent, TypesRepository } from 'services/code'
import { useStrings } from 'framework/strings'
import { useShowRequestError } from 'hooks/useShowRequestError'
import { decodeGitContent, isRefATag } from 'utils/GitUtils'
import { PlainButton } from 'components/PlainButton/PlainButton'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { permissionProps } from 'utils/Utils'
import css from './Readme.module.scss'

interface FolderContentProps {
  metadata: TypesRepository
  gitRef?: string
  readmeInfo: OpenapiContentInfo
  contentOnly?: boolean
  maxWidth?: string
}

function ReadmeViewer({ metadata, gitRef, readmeInfo, contentOnly, maxWidth }: FolderContentProps) {
  const { getString } = useStrings()
  const history = useHistory()
  const { routes, standalone, hooks } = useAppContext()
  const space = useGetSpaceParam()
  const { data, error, loading, refetch } = useGet<OpenapiGetContentOutput>({
    path: `/api/v1/repos/${metadata.path}/+/content/${readmeInfo?.path}`,
    queryParams: {
      include_commit: false,
      git_ref: gitRef
    },
    lazy: true
  })
  const ref = useRef(gitRef as string)

  useShowRequestError(error)

  // Fix an issue where readmeInfo is old (its new data is being fetched) but gitRef is
  // changed, causing README fetching to be incorrect. An example of this issue is to have
  // two branches, one has README.md and the other has README. If you switch between the two
  // branches, the README fetch will be incorrect (404).
  useEffect(() => {
    if (gitRef === ref.current) {
      refetch()
    } else {
      ref.current = gitRef as string
    }
  }, [refetch, gitRef])

  const permPushResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY'
      },
      permissions: ['code_repo_push']
    },
    [space]
  )

  const permsFinal = useMemo(() => {
    const perms = permissionProps(permPushResult, standalone)
    if (gitRef && isRefATag(gitRef) && perms) {
      return { tooltip: perms.tooltip, disabled: true }
    }

    if (gitRef && isRefATag(gitRef)) {
      return { tooltip: getString('editNotAllowed'), disabled: true }
    } else if (perms?.disabled) {
      return { disabled: perms.disabled, tooltip: perms.tooltip }
    }
    return { disabled: (gitRef && isRefATag(gitRef)) || false, tooltip: undefined }
  }, [permPushResult, gitRef]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <Container
      className={cx(css.readmeContainer, { [css.contentOnly]: contentOnly })}
      background={Color.WHITE}
      style={{ '--max-width': maxWidth } as React.CSSProperties}>
      <Render when={!contentOnly}>
        <Layout.Horizontal padding="small" className={css.heading}>
          <Heading level={5}>{readmeInfo.name}</Heading>
          <FlexExpander />
          {loading && <Icon name="steps-spinner" color={Color.PRIMARY_7} />}
          <PlainButton
            withoutCurrentColor
            size={ButtonSize.SMALL}
            variation={ButtonVariation.TERTIARY}
            iconProps={{ size: 16 }}
            text={getString('edit')}
            icon="code-edit"
            tooltip={permsFinal.tooltip}
            disabled={permsFinal.disabled}
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
          <MarkdownViewer source={decodeGitContent((data?.content as RepoFileContent)?.data)} />
        </Container>
      </Render>
    </Container>
  )
}

export const Readme = React.memo(ReadmeViewer)
