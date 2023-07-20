import React from 'react'
import {
  Button,
  ButtonVariation,
  Card,
  Container,
  FormikForm,
  FormInput,
  Layout,
  Page,
  Text,
  useToaster
} from '@harness/uicore'
import { FontVariation } from '@harness/design-system'
import { useHistory } from 'react-router-dom'
import { Formik } from 'formik'
import * as Yup from 'yup'

import { useUpdateUser } from 'services/code'
import { getErrorMessage } from 'utils/Utils'
import { String, useStrings } from 'framework/strings'

import css from './ChangePassword.module.scss'

interface PasswordForm {
  newPassword: string
  confirmPassword: string
}

const ChangePassword = () => {
  const history = useHistory()
  const { getString } = useStrings()
  const { showError, showSuccess } = useToaster()

  const { mutate } = useUpdateUser({})

  const onSubmit = async (values: PasswordForm) => {
    try {
      await mutate({
        password: values.newPassword
      })

      showSuccess(getString('changePasswordSuccesfully'))
      history.goBack()
    } catch (error) {
      showError(getErrorMessage(error))
    }
  }

  return (
    <Container className={css.mainCtn}>
      <Page.Header title={getString('changePassword')} />
      <Page.Body className={css.pageCtn}>
        <Card className={css.passwordCard}>
          <Formik<PasswordForm>
            initialValues={{
              newPassword: '',
              confirmPassword: ''
            }}
            validationSchema={Yup.object().shape({
              newPassword: Yup.string().required(getString('validation.newPasswordRequired')),
              confirmPassword: Yup.string()
                .test('passwords-match', getString('matchPassword'), function (value) {
                  return this.parent.newPassword === value
                })
                .required(getString('validation.confirmPasswordRequired'))
            })}
            onSubmit={onSubmit}>
            <FormikForm>
              <Layout.Vertical spacing="medium" width={320}>
                <Text font={{ variation: FontVariation.SMALL_SEMI }}>
                  <String useRichText stringID="enterNewPassword" />
                </Text>
                <FormInput.Text inputGroup={{ type: 'password' }} name="newPassword" />
                <Text font={{ variation: FontVariation.SMALL_SEMI }}>
                  <String useRichText stringID="confirmNewPassword" />
                </Text>
                <FormInput.Text inputGroup={{ type: 'password' }} name="confirmPassword" />
                <Layout.Horizontal spacing="medium">
                  <Button type="submit" text={getString('changePassword')} variation={ButtonVariation.PRIMARY} />
                  <Button text={getString('cancel')} variation={ButtonVariation.TERTIARY} onClick={history.goBack} />
                </Layout.Horizontal>
              </Layout.Vertical>
            </FormikForm>
          </Formik>
        </Card>
      </Page.Body>
    </Container>
  )
}

export default ChangePassword
