/*
 * Copyright 2021 Harness Inc. All rights reserved.
 * Use of this source code is governed by the PolyForm Shield 1.0.0 license
 * that can be found in the licenses directory at the root of this repository, also available at
 * https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
 */

import React, { useState } from 'react'
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
  FormInput
} from '@harness/uicore'
import cx from 'classnames'
import { FontVariation } from '@harness/design-system'
import { useMutate } from 'restful-react'
import { get } from 'lodash-es'
import { useModalHook } from '@harness/use-modal'
import { String, useStrings } from 'framework/strings'
import { DEFAULT_BRANCH_NAME, getErrorMessage, Unknown } from 'utils/Utils'
import type { TypesRepository, OpenapiCreateRepositoryRequest } from 'services/scm'
import { useAppContext } from 'AppContext'
import css from './CommitModalButton.module.scss'

enum CommitToGitRefOption {
  DIRECTLY = 'directly',
  NEW_BRANCH = 'new-branch'
}

interface RepoFormData {
  message: string
  extendedDescription: string
  newBranch: string
  branch: string
  commitToGitRefOption: CommitToGitRefOption
}

const formInitialValues: RepoFormData = {
  message: '',
  extendedDescription: '',
  newBranch: '',
  branch: '',
  commitToGitRefOption: CommitToGitRefOption.DIRECTLY
}

export interface CommitModalButtonProps extends Omit<ButtonProps, 'onClick' | 'onSubmit'> {
  gitRef: string
  commitMessagePlaceHolder: string
  resourcePath: string
  onSubmit: (data: TypesRepository) => void
}

export const CommitModalButton: React.FC<CommitModalButtonProps> = ({
  gitRef,
  resourcePath,
  commitMessagePlaceHolder,
  onSubmit,
  ...props
}) => {
  const ModalComponent: React.FC = () => {
    const { standalone } = useAppContext()
    const { getString } = useStrings()
    const [targetBranchOption, setTargetBranchOption] = useState(CommitToGitRefOption.DIRECTLY)
    const [branchName, setBranchName] = useState(gitRef)
    const { showError } = useToaster()
    const { mutate: createRepo, loading: submitLoading } = useMutate<TypesRepository>({
      verb: 'POST',
      path: `/api/v1/repos?spacePath=${gitRef}`
    })
    const loading = submitLoading

    console.log('Commit to', { targetBranchOption, branchName })

    const handleSubmit = (formData?: Unknown): void => {
      try {
        createRepo({
          message: branchName || get(formData, 'message', DEFAULT_BRANCH_NAME),
          extendedDescription: get(formData, 'extendedDescription', '').trim(),
          commitToGitRefOption: get(formData, 'commitToGitRefOption') === CommitToGitRefOption.DIRECTLY,
          uid: get(formData, 'name', '').trim(),
          parentId: standalone ? gitRef : 0
        } as OpenapiCreateRepositoryRequest)
          .then(response => {
            hideModal()
            onSubmit(response)
          })
          .catch(_error => {
            showError(getErrorMessage(_error), 0, 'failedToCreateRepo')
          })
      } catch (exception) {
        showError(getErrorMessage(exception), 0, 'failedToCreateRepo')
      }
    }

    return (
      <Dialog
        isOpen
        enforceFocus={false}
        onClose={hideModal}
        title={''}
        style={{ width: 700, maxHeight: '95vh', overflow: 'auto' }}>
        <Layout.Vertical className={cx(css.main)}>
          <Heading level={3} font={{ variation: FontVariation.H3 }} margin={{ bottom: 'xlarge' }}>
            {getString('commitChanges')}
          </Heading>

          <Container margin={{ right: 'xxlarge' }}>
            <Formik
              initialValues={formInitialValues}
              formName="commitChanges"
              enableReinitialize={true}
              validationSchema={yup.object().shape({})}
              validateOnChange
              validateOnBlur
              onSubmit={handleSubmit}>
              <FormikForm>
                <FormInput.Text
                  name="message"
                  label={getString('commitMessage')}
                  placeholder={commitMessagePlaceHolder}
                  tooltipProps={{
                    dataTooltipId: 'commitMessage'
                  }}
                  inputGroup={{ autoFocus: true }}
                />
                <FormInput.TextArea
                  className={css.extendedDescription}
                  name="extendedDescription"
                  placeholder={getString('optionalExtendedDescription')}
                />
                <Container
                  className={cx(
                    css.radioGroup,
                    targetBranchOption === CommitToGitRefOption.DIRECTLY ? css.directly : css.newBranch
                  )}>
                  <FormInput.RadioGroup
                    name="commitToGitRefOption"
                    label=""
                    onChange={e => {
                      setTargetBranchOption(get(e.target, 'defaultValue'))
                    }}
                    items={[
                      {
                        label: <String stringID="commitDirectlyTo" vars={{ gitRef }} useRichText />,
                        value: CommitToGitRefOption.DIRECTLY
                      },
                      {
                        label: <String stringID="commitToNewBranch" useRichText />,
                        value: CommitToGitRefOption.NEW_BRANCH
                      }
                    ]}
                  />
                  {targetBranchOption === CommitToGitRefOption.NEW_BRANCH && (
                    <Container>
                      <Layout.Horizontal spacing="medium" className={css.newBranchContainer}>
                        <Icon name="git-branch" />
                        <FormInput.Text
                          name="newBranch"
                          placeholder={getString('enterNewBranchName')}
                          tooltipProps={{
                            dataTooltipId: 'enterNewBranchName'
                          }}
                          inputGroup={{ autoFocus: true }}
                        />
                      </Layout.Horizontal>
                    </Container>
                  )}
                </Container>

                <Layout.Horizontal
                  spacing="small"
                  padding={{ right: 'xxlarge', top: 'xxlarge', bottom: 'large' }}
                  style={{ alignItems: 'center' }}>
                  <Button type="submit" text={getString('commitChanges')} intent={Intent.PRIMARY} disabled={loading} />
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
    onSubmit,
    gitRef,
    resourcePath,
    commitMessagePlaceHolder
  ])

  return <Button onClick={openModal} {...props} />
}
