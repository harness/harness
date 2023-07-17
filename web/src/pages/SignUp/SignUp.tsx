import React, { useCallback } from 'react'
import {
  Button,
  Color,
  Container,
  FormInput,
  Formik,
  FormikForm,
  Layout,
  StringSubstitute,
  Text,
  useToaster
} from '@harness/uicore'
import { Link, useHistory } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import AuthLayout from 'components/AuthLayout/AuthLayout'
import { routes } from 'RouteDefinitions'
import { getErrorMessage, type RegisterForm } from 'utils/Utils'
import { useOnRegister } from 'services/code'
import { useAPIToken } from 'hooks/useAPIToken'
import css from './SignUp.module.scss'

// Renders the Register page.
export const SignUp: React.FC = () => {
  const { getString } = useStrings()
  const history = useHistory()
  const { showError, showSuccess } = useToaster()
  const [, setToken] = useAPIToken()

  const { mutate } = useOnRegister({})
  const onRegister = useCallback(
    (data: RegisterForm) => {
      const formData = new FormData()
      formData.append('username', data.username)
      formData.append('email', data.username)
      formData.append('displayname', data.username)
      formData.append('password', data.password)

      mutate(formData as unknown as void)
        .then(result => {
          setToken(result.access_token as string)
          showSuccess('User was created')
          history.replace(routes.toSignIn())
        })
        .catch(error => {
          showError(getErrorMessage(error))
        })
    },
    [mutate, setToken, showSuccess, showError, history]
  )

  const handleSubmit = (data: RegisterForm): void => {
    if (data.username && data.password) {
      onRegister(data)
    }
  }
  return (
    <AuthLayout>
      <Container className={css.signUpContainer}>
        {/* <Container flex={{ justifyContent: 'space-between', alignItems: 'center' }} margin={{ bottom: 'xxxlarge' }}>
      <HarnessLogo height={25} />
    </Container> */}
        <Text font={{ size: 'large', weight: 'bold' }} color={Color.BLACK}>
          {getString('signUp')}
        </Text>
        {/* <Text font={{ size: 'medium' }} color={Color.GREY_500} margin={{ top: 'xsmall' }}>
      and get ship done.
    </Text> */}

        <Container margin={{ top: 'xxlarge' }}>
          <Formik<RegisterForm>
            initialValues={{ username: '', email: '', password: '' }}
            formName="loginPageForm"
            onSubmit={handleSubmit}>
            <FormikForm>
              <FormInput.Text
                placeholder={getString('enterUser')}
                name="username"
                label={getString('userId')}
                disabled={false}
              />
              <FormInput.Text placeholder={'email@work.com'} name="email" label={getString('email')} disabled={false} />

              <FormInput.Text
                name="password"
                label={getString('password')}
                inputGroup={{ type: 'password' }}
                disabled={false}
                placeholder={getString('characterLimit')}
              />
              <Button type="submit" intent="primary" loading={false} disabled={false} width="100%">
                {getString('signUp')}
              </Button>
            </FormikForm>
          </Formik>
        </Container>

        <Layout.Horizontal margin={{ top: 'xxxlarge' }} spacing="xsmall">
          <Text>
            <StringSubstitute
              str={getString('bySigningIn')}
              vars={{ policy: <a> {getString('privacyPolicy')} </a>, terms: <a> {getString('termsOfUse')} </a> }}
            />
          </Text>
        </Layout.Horizontal>
        <Layout.Horizontal margin={{ top: 'xxxlarge' }} spacing="xsmall">
          <Text>{getString('alreadyHaveAccount')}</Text>
          <Link to={routes.toSignIn()}>{getString('signIn')}</Link>
        </Layout.Horizontal>
      </Container>
    </AuthLayout>
  )
}
