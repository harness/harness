import {
  useToaster,
  type ButtonProps,
  Button,
  Dialog,
  Layout,
  Container,
  Formik,
  FormikForm,
  FormInput,
  FlexExpander,
  Text,
  Checkbox
} from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { FontVariation, Intent } from '@harnessio/design-system'
import React from 'react'
import { useMutate } from 'restful-react'
import * as yup from 'yup'
import { useModalHook } from 'hooks/useModalHook'
import { useStrings } from 'framework/strings'
import type { EnumTriggerAction, OpenapiCreateTriggerRequest, TypesTrigger } from 'services/code'
import { getErrorMessage } from 'utils/Utils'
import { triggerActions } from 'components/PipelineTriggersTab/PipelineTriggersTab'
import css from './NewTriggerModalButton.module.scss'

export interface TriggerFormData {
  name: string
  actions: EnumTriggerAction[]
}

const formInitialValues: TriggerFormData = {
  name: '',
  actions: []
}

export interface NewTriggerModalButtonProps extends Omit<ButtonProps, 'onClick' | 'onSubmit'> {
  repoPath: string
  pipeline: string
  modalTitle: string
  submitButtonTitle?: string
  cancelButtonTitle?: string
  onSuccess: () => void
}

export const NewTriggerModalButton: React.FC<NewTriggerModalButtonProps> = ({
  repoPath,
  pipeline,
  modalTitle,
  submitButtonTitle,
  cancelButtonTitle,
  onSuccess,
  ...props
}) => {
  const ModalComponent: React.FC = () => {
    const { getString } = useStrings()
    const { showError, showSuccess } = useToaster()

    const { mutate: createTrigger, loading } = useMutate<TypesTrigger>({
      verb: 'POST',
      path: `/api/v1/repos/${repoPath}/+/pipelines/${pipeline}/triggers`
    })

    const handleSubmit = async (formData: TriggerFormData) => {
      try {
        const payload: OpenapiCreateTriggerRequest = {
          actions: formData.actions,
          uid: formData.name
        }
        await createTrigger(payload)
        hideModal()
        showSuccess(getString('triggers.createSuccess'))
        onSuccess()
      } catch (exception) {
        showError(getErrorMessage(exception), 0, getString('triggers.failedToCreate'))
      }
    }

    return (
      <Dialog
        isOpen
        enforceFocus={false}
        onClose={hideModal}
        title={modalTitle}
        style={{ width: 700, maxHeight: '95vh', overflow: 'auto' }}>
        <Layout.Vertical padding={'large'} style={{ height: '100%' }} data-testid="add-trigger-modal">
          <Container>
            <Formik
              initialValues={formInitialValues}
              formName="addTrigger"
              enableReinitialize={true}
              validationSchema={yup.object().shape({
                name: yup
                  .string()
                  .required('name is required')
                  .matches(
                    /^[a-zA-Z_][a-zA-Z0-9-_.]*$/,
                    'name must start with a letter or _ and only contain [a-zA-Z0-9-_.]'
                  ),
                actions: yup.array().of(yup.string())
              })}
              validateOnChange
              validateOnBlur
              onSubmit={handleSubmit}>
              {formik => (
                <FormikForm>
                  <FormInput.Text
                    name="name"
                    label={getString('name')}
                    placeholder={getString('triggers.enterTriggerName')}
                    inputGroup={{ autoFocus: true }}
                  />
                  <Text font={{ variation: FontVariation.FORM_LABEL }} margin={{ bottom: 'xsmall' }}>
                    {getString('triggers.actions')}
                  </Text>
                  <Container className={css.actionsContainer} padding={'large'}>
                    {triggerActions.map(action => (
                      <Checkbox
                        key={action.name}
                        name="actions"
                        label={action.name}
                        value={action.value}
                        onChange={event => {
                          if (event.currentTarget.checked) {
                            formik.setFieldValue('actions', [...formik.values.actions, action.value])
                          } else {
                            formik.setFieldValue(
                              'actions',
                              formik.values.actions.filter((value: string) => value !== action.value)
                            )
                          }
                        }}
                      />
                    ))}
                  </Container>
                  <Layout.Horizontal spacing="small" padding={{ top: 'large' }} style={{ alignItems: 'center' }}>
                    <Button type="submit" text={getString('create')} intent={Intent.PRIMARY} disabled={loading} />
                    <Button text={cancelButtonTitle || getString('cancel')} minimal onClick={hideModal} />
                    <FlexExpander />
                    {loading && <Icon intent={Intent.PRIMARY} name="steps-spinner" size={16} />}
                  </Layout.Horizontal>
                </FormikForm>
              )}
            </Formik>
          </Container>
        </Layout.Vertical>
      </Dialog>
    )
  }

  const [openModal, hideModal] = useModalHook(ModalComponent, [onSuccess])

  return <Button onClick={openModal} {...props} />
}
