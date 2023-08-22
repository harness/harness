import React, { useCallback } from 'react'
import {
  Button,
  Container,
  FormInput,
  Formik,
  FormikForm,
  Layout,
  StringSubstitute,
  Text,
  useToaster
} from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import * as Yup from 'yup'
import { Link } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import AuthLayout from 'components/AuthLayout/AuthLayout'
import { useAppContext } from 'AppContext'
import { getErrorMessage, type RegisterForm } from 'utils/Utils'
import { useOnRegister } from 'services/code'
import { useAPIToken } from 'hooks/useAPIToken'
import css from './SignUp.module.scss'

// Renders the Register page.
export const SignUp: React.FC = () => {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const { showError, showSuccess } = useToaster()
  const [, setToken] = useAPIToken()

  const { mutate } = useOnRegister({})
  const onRegister = useCallback(
    (data: RegisterForm) => {
      mutate(
        {
          display_name: data.username,
          email: data.email,
          uid: data.username,
          password: data.password
        },
        {
          headers: { Authorization: '' }
        }
      )
        .then(result => {
          setToken(result.access_token as string)
          showSuccess(getString('userCreated'))
          window.location.replace(window.location.origin + routes.toCODEHome())
        })
        .catch(error => {
          showError(getErrorMessage(error))
        })
    },
    [mutate, setToken, showSuccess, showError, getString]
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

        <Container margin={{ top: 'xlarge' }}>
          <Formik<RegisterForm>
            initialValues={{ username: '', email: '', password: '', confirmPassword: '' }}
            formName="loginPageForm"
            validationSchema={Yup.object().shape({
              username: Yup.string().required(getString('userNameRequired')),
              email: Yup.string().email().required(getString('emailRequired')),
              password: Yup.string().min(6, getString('minPassLimit')).required(getString('passwordRequired')),
              confirmPassword: Yup.string()
                .required(getString('confirmPassRequired'))
                .oneOf([Yup.ref('password')], getString('matchPassword'))
            })}
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
              <FormInput.Text
                name="confirmPassword"
                label={getString('confirmPassword')}
                inputGroup={{ type: 'password' }}
                disabled={false}
                placeholder={getString('confirmPassword')}
              />

              <Button type="submit" intent="primary" loading={false} disabled={false} width="100%">
                {getString('signUp')}
              </Button>
            </FormikForm>
          </Formik>
        </Container>

        <Layout.Horizontal margin={{ top: 'xxlarge' }} spacing="xsmall">
          <Text>
            <StringSubstitute
              str={getString('bySigningIn')}
              vars={{ policy: <a> {getString('privacyPolicy')} </a>, terms: <a> {getString('termsOfUse')} </a> }}
            />
          </Text>
        </Layout.Horizontal>
        <Layout.Horizontal margin={{ top: 'xxlarge' }} spacing="xsmall">
          <Text>{getString('alreadyHaveAccount')}</Text>
          <Link to={routes.toSignIn()}>{getString('signIn')}</Link>
        </Layout.Horizontal>
      </Container>
    </AuthLayout>
  )
}
