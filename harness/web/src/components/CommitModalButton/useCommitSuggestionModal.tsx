/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useCallback, useState } from 'react'
import { Dialog, Intent } from '@blueprintjs/core'
import {
  Button,
  Container,
  Layout,
  FlexExpander,
  Formik,
  FormikForm,
  Heading,
  useToaster,
  FormInput,
  ButtonVariation
} from '@harnessio/uicore'
import { useAtomValue } from 'jotai'
import { Icon } from '@harnessio/icons'
import { useMutate } from 'restful-react'
import cx from 'classnames'
import { FontVariation } from '@harnessio/design-system'
import { useModalHook } from 'hooks/useModalHook'
import { useStrings } from 'framework/strings'
import { pullReqAtom } from 'pages/PullRequest/useGetPullRequestInfo'
import { repoMetadataAtom } from 'atoms/repoMetadata'
import css from './CommitModalButton.module.scss'

interface FormData {
  commitMessage?: string
  extendedDescription?: string
}

interface CommitModalProps extends FormData {
  title?: string
  onCommit: (formData: FormData) => Promise<Nullable<string>>
}

export function useCommitSuggestionsModal({
  title = '',
  commitMessage = '',
  extendedDescription = '',
  onCommit
}: CommitModalProps) {
  const ModalComponent: React.FC = () => {
    const { getString } = useStrings()
    const { showError } = useToaster()
    const [loading, setLoading] = useState(false)
    const onSubmit = useCallback(
      async (formData: FormData) => {
        setLoading(true)
        const error = await onCommit({
          commitMessage: formData.commitMessage || '',
          extendedDescription: formData.extendedDescription || ''
        })
        setLoading(false)

        if (error) {
          showError(error)
        } else {
          hideModal()
        }
      },
      [showError]
    )

    return (
      <Dialog
        isOpen
        enforceFocus={false}
        onClose={hideModal}
        title={''}
        style={{ width: 700, maxHeight: '95vh', overflow: 'auto' }}>
        <Layout.Vertical className={cx(css.main)}>
          <Heading level={3} font={{ variation: FontVariation.H3 }} margin={{ bottom: 'xlarge' }}>
            {title || getString('commitChanges')}
          </Heading>

          <Container margin={{ right: 'xxlarge' }} className={css.formContainer}>
            <Formik<FormData>
              initialValues={{
                commitMessage,
                extendedDescription
              }}
              formName="commitChanges"
              enableReinitialize={true}
              validateOnChange
              validateOnBlur
              onSubmit={onSubmit}>
              <FormikForm>
                <FormInput.Text
                  name="commitMessage"
                  label={getString('commitMessage')}
                  placeholder={commitMessage}
                  inputGroup={{ autoFocus: true }}
                />
                <FormInput.TextArea
                  className={css.extendedDescription}
                  name="extendedDescription"
                  placeholder={extendedDescription || getString('optionalExtendedDescription')}
                />

                <Layout.Horizontal spacing="small" padding={{ top: 'xxlarge', bottom: 'large' }}>
                  <Button
                    type="submit"
                    variation={ButtonVariation.PRIMARY}
                    text={getString('commit')}
                    disabled={loading}
                  />
                  <Button text={getString('cancel')} variation={ButtonVariation.LINK} onClick={hideModal} />
                  <FlexExpander />

                  {loading && <Icon intent={Intent.PRIMARY} name="steps-spinner" size={16} />}
                </Layout.Horizontal>
              </FormikForm>
            </Formik>
          </Container>
        </Layout.Vertical>
      </Dialog>
    )
  }

  const [openModal, hideModal] = useModalHook(ModalComponent, [])

  return [openModal, hideModal]
}

export function useCommitPullReqSuggestions() {
  const repoMetadata = useAtomValue(repoMetadataAtom)
  const pullReq = useAtomValue(pullReqAtom)
  const { mutate } = useMutate({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata?.path}/+/pullreq/${pullReq?.number}/comments/apply-suggestions`
  })

  return mutate
}
