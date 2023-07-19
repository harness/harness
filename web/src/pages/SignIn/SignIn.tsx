import React, { useCallback } from 'react'
import { Button, Color, Container, FormInput, Formik, FormikForm, Layout, Text, useToaster } from '@harness/uicore'
import * as Yup from 'yup'

import { Link, useHistory } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { useOnLogin } from 'services/code'
import AuthLayout from 'components/AuthLayout/AuthLayout'
import { routes } from 'RouteDefinitions'
import { useAPIToken } from 'hooks/useAPIToken'
import { getErrorMessage, type LoginForm } from 'utils/Utils'
import css from './SignIn.module.scss'

export const SignIn: React.FC = () => {
  const { getString } = useStrings()
  const history = useHistory()
  const [, setToken] = useAPIToken()
  const { mutate } = useOnLogin({})
  const { showError } = useToaster()

  const onLogin = useCallback(
    (data: LoginForm) => {
      mutate({ login_identifier: data.username, password: data.password } as unknown as void)
        .then(result => {
          setToken(result.access_token as string)
          history.replace(routes.toCODESpaces())
        })

        .catch(error => {
          showError(getErrorMessage(error))
        })
    },
    [mutate, history, setToken, showError]
  )

  const handleSubmit = (data: LoginForm): void => {
    if (data.username && data.password) {
      onLogin(data)
    }
  }
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
            onSubmit={handleSubmit}
            validationSchema={Yup.object().shape({
              username: Yup.string().required(getString('userNameRequired')),

              password: Yup.string().min(6).required(getString('passwordRequired'))
            })}>
            <FormikForm>
              <FormInput.Text name="username" label={getString('email')} disabled={false} />
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
