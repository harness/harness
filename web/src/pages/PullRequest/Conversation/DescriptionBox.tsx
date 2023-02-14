import React, { useState } from 'react'
import { Container, useToaster } from '@harness/uicore'
import cx from 'classnames'
import { useMutate } from 'restful-react'
import { MarkdownViewer } from 'components/SourceCodeViewer/SourceCodeViewer'
import { useStrings } from 'framework/strings'
import type { OpenapiUpdatePullReqRequest } from 'services/code'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { MarkdownEditorWithPreview } from 'components/MarkdownEditorWithPreview/MarkdownEditorWithPreview'
import { getErrorMessage } from 'utils/Utils'
import type { ConversationProps } from './Conversation'
import css from './Conversation.module.scss'

export const DescriptionBox: React.FC<ConversationProps> = ({
  repoMetadata,
  pullRequestMetadata,
  onCommentUpdate: refreshPullRequestMetadata
}) => {
  const [edit, setEdit] = useState(false)
  // const [updated, setUpdated] = useState(pullRequestMetadata.edited as number)
  const [originalContent, setOriginalContent] = useState(pullRequestMetadata.description as string)
  const [content, setContent] = useState(originalContent)
  const { getString } = useStrings()
  const { showError } = useToaster()
  const { mutate } = useMutate({
    verb: 'PATCH',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata.number}`
  })

  return (
    <Container className={cx(css.box, css.desc)}>
      <Container padding={{ left: 'small', bottom: 'small' }}>
        {(edit && (
          <MarkdownEditorWithPreview
            value={content}
            onSave={value => {
              const payload: OpenapiUpdatePullReqRequest = {
                title: pullRequestMetadata.title,
                description: value
              }
              mutate(payload)
                .then(() => {
                  setContent(value)
                  setOriginalContent(value)
                  setEdit(false)
                  // setUpdated(Date.now())
                  refreshPullRequestMetadata()
                })
                .catch(exception => showError(getErrorMessage(exception), 0, getString('pr.failedToUpdate')))
            }}
            onCancel={() => {
              setContent(originalContent)
              setEdit(false)
            }}
            i18n={{
              placeHolder: getString('pr.enterDesc'),
              tabEdit: getString('write'),
              tabPreview: getString('preview'),
              save: getString('save'),
              cancel: getString('cancel')
            }}
            maxEditorHeight="400px"
          />
        )) || (
          <Container className={css.mdWrapper}>
            <MarkdownViewer source={content} />
            <Container className={css.menuWrapper}>
              <OptionsMenuButton
                isDark={true}
                icon="Options"
                iconProps={{ size: 14 }}
                style={{ padding: '5px' }}
                items={[
                  {
                    text: getString('edit'),
                    className: css.optionMenuIcon,
                    hasIcon: true,
                    iconName: 'Edit',
                    onClick: () => setEdit(true)
                  }
                ]}
              />
            </Container>
          </Container>
        )}
      </Container>
    </Container>
  )
}
