import React, { useCallback } from 'react'
import { Button, Container, FormInput, Formik, FormikForm, Layout, Text, useToaster } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import * as Yup from 'yup'
import { Link } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { useOnLogin } from 'services/code'
import AuthLayout from 'components/AuthLayout/AuthLayout'
import { useAppContext } from 'AppContext'
import { getErrorMessage, type LoginForm } from 'utils/Utils'
import css from './SignIn.module.scss'

export const SignIn: React.FC = () => {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const { mutate } = useOnLogin({
    queryParams: {
      include_cookie: true
    }
  })
  const { showError } = useToaster()
  const onLogin = useCallback(
    ({ username, password }: LoginForm) => {
      mutate(
        { login_identifier: username, password },
        {
          headers: { Authorization: '' }
        }
      )
        .then(() => {
          window.location.replace(window.location.origin + routes.toCODEHome())
        })
        .catch(error => {
          showError(getErrorMessage(error))
        })
    },
    [mutate, showError, routes]
  )
  const onSubmit = useCallback(
    (data: LoginForm): void => {
      if (data.username && data.password) {
        onLogin(data)
      }
    },
    [onLogin]
  )

  return (
    <AuthLayout>
      <Container className={css.signInContainer}>
        <Text font={{ size: 'large', weight: 'bold' }} color={Color.BLACK}>
          {getString('signIn')}
        </Text>

        <Container margin={{ top: 'xxlarge' }}>
          <Formik<LoginForm>
            initialValues={{ username: '', password: '' }}
            formName="loginPageForm"
            onSubmit={onSubmit}
            validationSchema={Yup.object().shape({
              username: Yup.string().required(getString('userNameRequired')),
              password: Yup.string().required(getString('passwordRequired'))
            })}>
            <FormikForm>
              <FormInput.Text name="username" label={getString('emailUser')} disabled={false} />
              <FormInput.Text
                name="password"
                label={getString('password')}
                inputGroup={{ type: 'password' }}
                disabled={false}
              />
              <Button type="submit" intent="primary" loading={false} disabled={false} width="100%">
                {getString('signIn')}
              </Button>
            </FormikForm>
          </Formik>
        </Container>

        <Layout.Horizontal margin={{ top: 'xxxlarge' }} spacing="xsmall">
          <Text>{getString('noAccount?')}</Text>
          <Link to={routes.toRegister()}>{getString('signUp')}</Link>
        </Layout.Horizontal>
      </Container>
    </AuthLayout>
  )
}
