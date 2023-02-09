/*
 * Copyright 2021 Harness Inc. All rights reserved.
 * Use of this source code is governed by the PolyForm Shield 1.0.0 license
 * that can be found in the licenses directory at the root of this repository, also available at
 * https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
 */

import React from 'react'
import { Dialog, Intent } from '@blueprintjs/core'
import * as yup from 'yup'
import { Render } from 'react-jsx-match'
import {
  Button,
  ButtonProps,
  Container,
  Layout,
  FlexExpander,
  Icon,
  Formik,
  FormikForm,
  Heading,
  useToaster,
  FormInput,
  ButtonVariation
} from '@harness/uicore'
import { Color, FontVariation } from '@harness/design-system'
import { useMutate } from 'restful-react'
import { get } from 'lodash-es'
import { useModalHook } from '@harness/use-modal'
import { useStrings } from 'framework/strings'
import { getErrorMessage } from 'utils/Utils'
import type { OpenapiCreatePullReqRequest, TypesPullReq } from 'services/code'
import { CodeIcon, GitInfoProps } from 'utils/GitUtils'
import css from './CreatePullRequestModal.module.scss'

interface FormData {
  title: string
  description: string
  draft: boolean
}

interface CreatePullRequestModalProps extends Pick<GitInfoProps, 'repoMetadata'> {
  targetGitRef: string
  sourceGitRef: string
  onSuccess: (data: TypesPullReq) => void
}

interface CreatePullRequestModalButtonProps extends Omit<ButtonProps, 'onClick'>, CreatePullRequestModalProps {}

export function useCreatePullRequestModal({
  repoMetadata,
  targetGitRef,
  sourceGitRef,
  onSuccess
}: CreatePullRequestModalProps) {
  const ModalComponent: React.FC = () => {
    const { getString } = useStrings()
    const { showError } = useToaster()
    const { mutate: createPullRequest, loading } = useMutate<TypesPullReq>({
      verb: 'POST',
      path: `/api/v1/repos/${repoMetadata.path}/+/pullreq`
    })
    const handleSubmit = (formData: FormData) => {
      const title = get(formData, 'title', '').trim()
      const description = get(formData, 'description', '').trim()

      const pullReqUrl = window.location.href.split('compare')?.[0]
      const payload: OpenapiCreatePullReqRequest = {
        target_branch: targetGitRef,
        source_branch: sourceGitRef,
        title: title,
        description: description,
        is_draft: formData.draft
      }

      try {
        createPullRequest(payload)
          .then(response => {
            hideModal()
            onSuccess(response)
          })
          .catch(_error => {
            if (_error.status === 409) {
              showError(
                <>
                  {getString('pullRequest')}
                  <strong>
                    <a
                      className={css.hyperlink}
                      color={Color.PRIMARY_7}
                      href={`${pullReqUrl}${_error.data.values.number}`}>
                      {` #${_error.data.values.number} ${_error.data.values.title} `}
                    </a>
                  </strong>
                  {getString('alreadyExists')}
                </>
              )
            } else {
              showError(getErrorMessage(_error), 0, 'pr.failedToCreate')
            }
          })
      } catch (exception) {
        showError(getErrorMessage(exception), 0, 'pr.failedToCreate')
      }
    }

    return (
      <Dialog
        isOpen
        enforceFocus={false}
        onClose={hideModal}
        title={''}
        style={{ width: '60vw', maxHeight: '95vh', overflow: 'auto' }}>
        <Layout.Vertical padding={{ left: 'xxlarge' }} style={{ height: '100%' }} className={css.main}>
          <Heading className={css.title} font={{ variation: FontVariation.H3 }} margin={{ bottom: 'xlarge' }}>
            <Icon name={CodeIcon.PullRequest} size={22} /> {getString('pr.modalTitle')}
          </Heading>
          <Container margin={{ right: 'xxlarge' }}>
            <Formik<FormData>
              initialValues={{
                title: '',
                description: '',
                draft: false
              }}
              formName="createPullRequest"
              enableReinitialize={true}
              validationSchema={yup.object().shape({
                title: yup.string().trim().required(),
                description: yup.string().trim().required()
              })}
              validateOnChange
              validateOnBlur
              onSubmit={handleSubmit}>
              <FormikForm>
                <FormInput.Text
                  name="title"
                  label={getString('title')}
                  placeholder={getString('pr.titlePlaceHolder')}
                  tooltipProps={{
                    dataTooltipId: 'createPullRequestTitle'
                  }}
                  inputGroup={{ autoFocus: true }}
                />

                <FormInput.TextArea
                  name="description"
                  label={getString('description')}
                  placeholder={getString('pr.descriptionPlaceHolder')}
                  tooltipProps={{
                    dataTooltipId: 'createPullRequestDescription'
                  }}
                  className={css.description}
                  maxLength={1024 * 50}
                />
                <FormInput.CheckBox label={getString('pr.createDraftPR')} name="draft" className={css.checkbox} />

                <Layout.Horizontal
                  spacing="small"
                  padding={{ right: 'xxlarge', top: 'xlarge', bottom: 'large' }}
                  style={{ alignItems: 'center' }}>
                  <Button
                    type="submit"
                    text={getString('createPullRequest')}
                    variation={ButtonVariation.PRIMARY}
                    disabled={loading}
                  />
                  <Button text={getString('cancel')} variation={ButtonVariation.LINK} onClick={hideModal} />
                  <FlexExpander />

                  <Render when={loading}>
                    <Icon intent={Intent.PRIMARY} name="spinner" size={16} />
                  </Render>
                </Layout.Horizontal>
              </FormikForm>
            </Formik>
          </Container>
        </Layout.Vertical>
      </Dialog>
    )
  }
  const [openModal, hideModal] = useModalHook(ModalComponent, [onSuccess])

  return openModal
}

export const CreatePullRequestModalButton: React.FC<CreatePullRequestModalButtonProps> = ({
  onSuccess,
  repoMetadata,
  targetGitRef,
  sourceGitRef,
  ...props
}) => {
  const openModal = useCreatePullRequestModal({ repoMetadata, targetGitRef, sourceGitRef, onSuccess })
  return <Button onClick={() => openModal()} {...props} />
}
