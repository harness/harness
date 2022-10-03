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
  FormInput,
  Text
} from '@harness/uicore'
import { FontVariation } from '@harness/design-system'
import { useModalHook } from '@harness/use-modal'
import { useStrings } from 'framework/strings'
import { getErrorMessage, Unknown } from 'utils/Utils'

export interface NewRepoModalButtonProps extends Omit<ButtonProps, 'onClick' | 'onSubmit'> {
  accountIdentifier: string
  orgIdentifier: string
  projectIdentifier: string

  modalTitle: string
  submitButtonTitle?: string
  cancelButtonTitle?: string

  onSubmit: (data: Unknown) => void
}

export const NewRepoModalButton: React.FC<NewRepoModalButtonProps> = ({
  accountIdentifier,
  orgIdentifier,
  projectIdentifier,
  modalTitle,
  submitButtonTitle,
  cancelButtonTitle,
  onSubmit,
  ...props
}) => {
  const ModalComponent: React.FC = () => {
    const { getString } = useStrings()
    const { showError } = useToaster()

    const [submitLoading, setSubmitLoading] = useState(false)
    const handleSubmit = (_data?: Unknown): void => {
      setSubmitLoading(true)
      try {
        onSubmit({})
      } catch (exception) {
        setSubmitLoading(false)
        showError(getErrorMessage(exception), 0, 'cf.save.ff.error')
      }
    }

    const loading = submitLoading

    return (
      <Dialog isOpen enforceFocus={false} onClose={hideModal} title={''} style={{ width: 700, maxHeight: '95vh' }}>
        <Layout.Vertical
          padding={{ left: 'xxlarge' }}
          style={{ height: '100%' }}
          data-testid="add-target-to-flag-modal">
          <Heading level={3} font={{ variation: FontVariation.H3 }} margin={{ bottom: 'xlarge' }}>
            {modalTitle}
          </Heading>

          <Container margin={{ right: 'xxlarge' }}>
            <Formik
              initialValues={{
                gitDetails: '',
                autoCommit: ''
              }}
              formName="editVariations"
              enableReinitialize={true}
              validationSchema={yup.object().shape({
                // gitDetails: 'gitSyncFormMeta?.gitSyncValidationSchema'
              })}
              validateOnChange
              validateOnBlur
              onSubmit={handleSubmit}>
              <FormikForm>
                <FormInput.Text
                  name="name"
                  label="Name"
                  placeholder="Enter Repository Name"
                  tooltipProps={{
                    dataTooltipId: 'repositoryNameTextField'
                  }}
                  inputGroup={{ autoFocus: true }}
                />
                <FormInput.Text
                  name="description"
                  label="Description"
                  placeholder="Enter a description (optional)"
                  tooltipProps={{
                    dataTooltipId: 'repositoryDescriptionTextField'
                  }}
                />
                <Container margin={{ top: 'medium', bottom: 'xlarge' }}>
                  <Text>
                    Your repository will be initialized with a <strong>main</strong> branch.
                  </Text>
                </Container>

                <FormInput.Select
                  name="license"
                  label="Add License"
                  placeholder="None"
                  items={[
                    { label: 'Red', value: 'red' },
                    { label: 'Blue', value: 'blue' },
                    {
                      label: 'TryingTryingTryingTryingTryingTryingTryingTryingTryingTryingTryingTryingTrying',
                      value: 'xyz'
                    },
                    { label: 'Trying a long phrase with spaces to try out different combinations', value: 'abcd' }
                  ]}
                />

                <FormInput.Select
                  name="gitignore"
                  label="Add a .gitignore"
                  placeholder="None"
                  items={[
                    { label: 'Red', value: 'red' },
                    { label: 'Blue', value: 'blue' },
                    {
                      label: 'TryingTryingTryingTryingTryingTryingTryingTryingTryingTryingTryingTryingTrying',
                      value: 'xyz'
                    },
                    { label: 'Trying a long phrase with spaces to try out different combinations', value: 'abcd' }
                  ]}
                />

                <FormInput.CheckBox
                  name="addReadme"
                  label="Add a README file"
                  tooltipProps={{
                    dataTooltipId: 'addReadMe'
                  }}
                />
                <Layout.Horizontal
                  spacing="small"
                  padding={{ right: 'xxlarge', top: 'xxxlarge', bottom: 'large' }}
                  style={{ alignItems: 'center' }}>
                  <Button type="submit" text={'Create Repository'} intent={Intent.PRIMARY} disabled={loading} />
                  <Button text={cancelButtonTitle || getString('cancel')} minimal onClick={hideModal} />
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

  const [openModal, hideModal] = useModalHook(ModalComponent, [onSubmit])

  return <Button onClick={openModal} {...props} />
}
