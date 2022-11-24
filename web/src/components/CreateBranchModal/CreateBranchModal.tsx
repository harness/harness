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
  Label
} from '@harness/uicore'
import { FontVariation } from '@harness/design-system'
import { useMutate } from 'restful-react'
import { get } from 'lodash-es'
import { useModalHook } from '@harness/use-modal'
import { useStrings } from 'framework/strings'
import { getErrorMessage } from 'utils/Utils'
import { CodeIcon, GitInfoProps, GitRefType, isGitBranchNameValid } from 'utils/GitUtils'
import { BranchTagSelect } from 'components/BranchTagSelect/BranchTagSelect'
import type { RepoBranch } from 'services/code'
import css from './CreateBranchModal.module.scss'

interface FormData {
  name: string
  sourceBranch: string
}

interface UseCreateBranchModalProps extends Pick<GitInfoProps, 'repoMetadata'> {
  suggestedBranchName?: string
  suggestedSourceBranch?: string
  onSuccess: (data: RepoBranch) => void
  showSuccessMessage?: boolean
}

interface CreateBranchModalButtonProps extends Omit<ButtonProps, 'onClick'>, Pick<GitInfoProps, 'repoMetadata'> {
  onSuccess: (data: RepoBranch) => void
  showSuccessMessage?: boolean
}

export function useCreateBranchModal({
  suggestedBranchName = '',
  suggestedSourceBranch = '',
  onSuccess,
  repoMetadata,
  showSuccessMessage
}: UseCreateBranchModalProps) {
  const [branchName, setBranchName] = useState(suggestedBranchName)
  const ModalComponent: React.FC = () => {
    const { getString } = useStrings()
    const [sourceBranch, setSourceBranch] = useState(suggestedSourceBranch || (repoMetadata.defaultBranch as string))
    const { showError, showSuccess } = useToaster()
    const { mutate: createBranch, loading } = useMutate<RepoBranch>({
      verb: 'POST',
      path: `/api/v1/repos/${repoMetadata.path}/+/branches`
    })
    const handleSubmit = (formData?: Unknown): void => {
      const name = get(formData, 'name').trim()
      try {
        createBranch({
          name,
          target: sourceBranch
        })
          .then(response => {
            hideModal()
            onSuccess(response)
            if (showSuccessMessage) {
              showSuccess(getString('branchCreated', { branch: name }), 5000)
            }
          })
          .catch(_error => {
            showError(getErrorMessage(_error), 0, 'failedToCreateBranch')
          })
      } catch (exception) {
        showError(getErrorMessage(exception), 0, 'failedToCreateBranch')
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
            <Icon name={CodeIcon.Branch} size={22} /> {getString('createABranch')}
          </Heading>
          <Container margin={{ right: 'xxlarge' }}>
            <Formik<FormData>
              initialValues={{
                name: branchName,
                sourceBranch: suggestedSourceBranch
              }}
              formName="createGitBranch"
              enableReinitialize={true}
              validationSchema={yup.object().shape({
                name: yup
                  .string()
                  .trim()
                  .required()
                  .test('valid-branch-name', getString('validation.gitBranchNameInvalid'), value => {
                    const val = value || ''
                    return !!val && isGitBranchNameValid(val)
                  })
              })}
              validateOnChange
              validateOnBlur
              onSubmit={handleSubmit}>
              <FormikForm>
                <FormInput.Text
                  name="name"
                  label={getString('branchName')}
                  placeholder={getString('nameYourBranch')}
                  tooltipProps={{
                    dataTooltipId: 'repositoryBranchTextField'
                  }}
                  inputGroup={{ autoFocus: true }}
                />
                <Container margin={{ top: 'medium', bottom: 'medium' }}>
                  <Label className={css.label}>{getString('branchSourceDesc')}</Label>
                  {/* <Text className={css.branchSourceDesc}>{getString('branchSourceDesc')}</Text> */}
                  <Layout.Horizontal spacing="medium" padding={{ top: 'xsmall' }}>
                    <BranchTagSelect
                      repoMetadata={repoMetadata}
                      disableBranchCreation
                      disableViewAllBranches
                      forBranchesOnly
                      gitRef={sourceBranch}
                      gitRefType={GitRefType.BRANCH}
                      onSelect={ref => {
                        setSourceBranch(ref)
                      }}
                    />
                    <FlexExpander />
                  </Layout.Horizontal>
                </Container>

                <Layout.Horizontal
                  spacing="small"
                  padding={{ right: 'xxlarge', top: 'xxxlarge', bottom: 'large' }}
                  style={{ alignItems: 'center' }}>
                  <Button type="submit" text={getString('createBranch')} intent={Intent.PRIMARY} disabled={loading} />
                  <Button text={getString('cancel')} minimal onClick={hideModal} />
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

export const CreateBranchModalButton: React.FC<CreateBranchModalButtonProps> = ({
  onSuccess,
  repoMetadata,
  showSuccessMessage,
  ...props
}) => {
  const openModal = useCreateBranchModal({ repoMetadata, onSuccess, showSuccessMessage })
  return <Button onClick={() => openModal()} {...props} />
}
