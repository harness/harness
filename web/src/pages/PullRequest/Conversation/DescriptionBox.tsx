import React, { useEffect, useState } from 'react'
import { Container, useToaster } from '@harnessio/uicore'
import cx from 'classnames'
import { useMutate } from 'restful-react'
import { MarkdownViewer } from 'components/MarkdownViewer/MarkdownViewer'
import { useStrings } from 'framework/strings'
import type { OpenapiUpdatePullReqRequest } from 'services/code'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { MarkdownEditorWithPreview } from 'components/MarkdownEditorWithPreview/MarkdownEditorWithPreview'
import { NavigationCheck } from 'components/NavigationCheck/NavigationCheck'
import { getErrorMessage } from 'utils/Utils'
import type { ConversationProps } from './Conversation'
import css from './Conversation.module.scss'

interface DescriptionBoxProps extends Omit<ConversationProps, 'onCancelEditDescription'> {
  onCancelEditDescription: () => void
}

export const DescriptionBox: React.FC<DescriptionBoxProps> = ({
  repoMetadata,
  pullRequestMetadata,
  onCommentUpdate: refreshPullRequestMetadata,
  onCancelEditDescription
}) => {
  const [edit, setEdit] = useState(false)
  const [dirty, setDirty] = useState(false)
  const [originalContent, setOriginalContent] = useState(pullRequestMetadata.description as string)
  const [content, setContent] = useState(originalContent)
  const { getString } = useStrings()
  const { showError } = useToaster()
  const { mutate } = useMutate({
    verb: 'PATCH',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata.number}`
  })

  useEffect(() => {
    setEdit(!pullRequestMetadata?.description?.length)
  }, [pullRequestMetadata?.description?.length])

  return (
    <Container className={cx({ [css.box]: !edit, [css.desc]: !edit })}>
      <Container padding={!edit ? { left: 'small', bottom: 'small' } : undefined}>
        {(edit && (
          <MarkdownEditorWithPreview
            value={content}
            onSave={value => {
              const payload: OpenapiUpdatePullReqRequest = {
                title: pullRequestMetadata.title,
                description: value || ''
              }
              mutate(payload)
                .then(() => {
                  setContent(value)
                  setOriginalContent(value)
                  setEdit(false)
                  refreshPullRequestMetadata()
                })
                .catch(exception => showError(getErrorMessage(exception), 0, getString('pr.failedToUpdate')))
            }}
            onCancel={() => {
              setContent(originalContent)
              setEdit(false)
              onCancelEditDescription()
            }}
            setDirty={setDirty}
            i18n={{
              placeHolder: getString('pr.enterDesc'),
              tabEdit: getString('write'),
              tabPreview: getString('preview'),
              save: getString('save'),
              cancel: getString('cancel')
            }}
            editorHeight="400px"
            autoFocusAndPositioning
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
      <NavigationCheck when={dirty} />
    </Container>
  )
}
