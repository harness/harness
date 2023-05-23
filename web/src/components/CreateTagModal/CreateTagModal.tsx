/*
 * Copyright 2021 Harness Inc. All rights reserved.
 * Use of this source code is governed by the PolyForm Shield 1.0.0 license
 * that can be found in the licenses directory at the root of this repository, also available at
 * https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
 */

import React, { useCallback, useState } from 'react'
import { Dialog, Intent } from '@blueprintjs/core'
import * as yup from 'yup'
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
  Label,
  ButtonVariation
} from '@harness/uicore'
import { FontVariation } from '@harness/design-system'
import { useMutate } from 'restful-react'
import { get } from 'lodash-es'
import { useModalHook } from '@harness/use-modal'
import { useStrings } from 'framework/strings'
import { getErrorMessage, permissionProps } from 'utils/Utils'
import { GitInfoProps, isGitBranchNameValid } from 'utils/GitUtils'
import { BranchTagSelect } from 'components/BranchTagSelect/BranchTagSelect'
import type { RepoBranch } from 'services/code'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useAppContext } from 'AppContext'
import css from './CreateTagModal.module.scss'

interface FormData {
  name: string
  sourceBranch: string
  description: string
}

interface UseCreateBranchModalProps extends Pick<GitInfoProps, 'repoMetadata'> {
  suggestedBranchName?: string
  suggestedSourceBranch?: string
  onSuccess: (data: RepoBranch) => void
  showSuccessMessage?: boolean
}

interface CreateBranchModalButtonProps extends Omit<ButtonProps, 'onClick'>, UseCreateBranchModalProps {
  onSuccess: (data: RepoBranch) => void
  showSuccessMessage?: boolean
}

export function useCreateTagModal({
  suggestedBranchName = '',
  suggestedSourceBranch = '',
  onSuccess,
  repoMetadata,
  showSuccessMessage
}: UseCreateBranchModalProps) {
  const [branchName, setBranchName] = useState(suggestedBranchName)
  const ModalComponent: React.FC = () => {
    const { getString } = useStrings()
    const [sourceBranch, setSourceBranch] = useState(suggestedSourceBranch || (repoMetadata.default_branch as string))
    const { showError, showSuccess } = useToaster()
    const { mutate: createTag, loading } = useMutate<RepoBranch>({
      verb: 'POST',
      path: `/api/v1/repos/${repoMetadata.path}/+/tags`
    })
    const handleSubmit = (formData: FormData) => {
      const name = get(formData, 'name').trim()
      const description = get(formData, 'description').trim()

      try {
        createTag({
          name,
          message: description,
          target: sourceBranch
        })
          .then(response => {
            hideModal()
            onSuccess(response)
            if (showSuccessMessage) {
              showSuccess(getString('tagCreated', { tag: name }), 5000)
            }
          })
          .catch(_error => {
            showError(getErrorMessage(_error), 0, 'failedToCreateTag')
          })
      } catch (exception) {
        showError(getErrorMessage(exception), 0, 'failedToCreateTag')
      }
    }

    return (
      <Dialog
        isOpen
        enforceFocus={false}
        onClose={hideModal}
        title={''}
        style={{ width: 700, maxHeight: '95vh', overflow: 'auto' }}>
        <Layout.Vertical padding={{ left: 'xxlarge' }} style={{ height: '100%' }} className={css.main}>
          <Heading className={css.title} font={{ variation: FontVariation.H3 }} margin={{ bottom: 'xlarge' }}>
            {getString('createATag')}
          </Heading>
          <Container margin={{ right: 'xxlarge' }}>
            <Formik<FormData>
              initialValues={{
                name: branchName,
                sourceBranch: suggestedSourceBranch,
                description: ''
              }}
              formName="createGitTag"
              enableReinitialize={true}
              validationSchema={yup.object().shape({
                name: yup
                  .string()
                  .trim()
                  .required()
                  .test('valid-tag-name', getString('validation.gitTagNameInvalid'), value => {
                    const val = value || ''
                    return !!val && isGitBranchNameValid(val)
                  }),
                description: yup.string().required()
              })}
              validateOnChange
              validateOnBlur
              onSubmit={handleSubmit}>
              <FormikForm>
                <FormInput.Text
                  name="name"
                  label={getString('name')}
                  placeholder={getString('enterTagPlaceholder')}
                  tooltipProps={{
                    dataTooltipId: 'repositoryTagTextField'
                  }}
                  inputGroup={{ autoFocus: true }}
                />
                <Container margin={{ top: 'medium', bottom: 'medium' }}>
                  <Label className={css.label}>{getString('basedOn')}</Label>
                  {/* <Text className={css.branchSourceDesc}>{getString('branchSourceDesc')}</Text> */}
                  <Layout.Horizontal spacing="medium" padding={{ top: 'xsmall' }}>
                    <BranchTagSelect
                      repoMetadata={repoMetadata}
                      disableBranchCreation
                      disableViewAllBranches
                      forBranchesOnly
                      gitRef={sourceBranch}
                      onSelect={setSourceBranch}
                    />
                    <FlexExpander />
                  </Layout.Horizontal>
                </Container>
                <FormInput.TextArea
                  label={getString('description')}
                  className={css.extendedDescription}
                  name="description"
                  placeholder={getString('tagDescription')}
                />

                <Layout.Horizontal
                  spacing="small"
                  padding={{ right: 'xxlarge', top: 'xxxlarge', bottom: 'large' }}
                  style={{ alignItems: 'center' }}>
                  <Button
                    type="submit"
                    text={getString('create')}
                    variation={ButtonVariation.PRIMARY}
                    disabled={loading}
                  />
                  <Button text={getString('cancel')} variation={ButtonVariation.LINK} onClick={hideModal} />
                  <FlexExpander />

                  {loading && <Icon intent={Intent.PRIMARY} name="spinner" size={16} />}
                </Layout.Horizontal>
              </FormikForm>
            </Formik>
          </Container>
        </Layout.Vertical>
      </Dialog>
    )
  }
  const [openModal, hideModal] = useModalHook(ModalComponent, [
    onSuccess,
    suggestedBranchName,
    suggestedSourceBranch,
    showSuccessMessage
  ])
  const fn = useCallback(
    (_branchName?: string) => {
      if (_branchName) {
        setBranchName(_branchName)
      }
      openModal()
    },
    [setBranchName, openModal]
  )

  return fn
}

export const CreateTagModalButton: React.FC<CreateBranchModalButtonProps> = ({
  onSuccess,
  repoMetadata,
  showSuccessMessage,
  ...props
}) => {
  const openModal = useCreateTagModal({ repoMetadata, onSuccess, showSuccessMessage })
  const { standalone } = useAppContext()
  const { hooks } = useAppContext()
  const space = useGetSpaceParam()

  const permPushResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY'
      },
      permissions: ['code_repo_push']
    },
    [space]
  )
  return <Button onClick={() => openModal()} {...props} {...permissionProps(permPushResult, standalone)} />
}
