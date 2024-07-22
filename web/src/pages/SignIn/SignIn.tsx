/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useState } from 'react'
// import {
//   Button,
//   Container,
//   FlexExpander,
//   FormInput,
//   Formik,
//   FormikForm,
//   Layout,
//   Text,
//   useToaster,
//   StringSubstitute
// } from '@harnessio/uicore'
import { Button, Card, CardContent, CardHeader, CardTitle, Input, Label } from '@harnessio/canary'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'

import Logo from 'components/canary/misc/logo-purple'
import bodyBlur from 'images/body-purple-blur.svg?url'
import FooterStrap from 'components/canary/layout/FooterStrap'
import Container from 'components/canary/layout/container'

// import { Color } from '@harnessio/design-system'
// import * as Yup from 'yup'
// import { Link } from 'react-router-dom'
// import { useStrings } from 'framework/strings'
// import { useOnLogin } from 'services/code'
// import AuthLayout from 'components/AuthLayout/AuthLayout'
// import { useAppContext } from 'AppContext'
// import { getErrorMessage, type LoginForm } from 'utils/Utils'
// import css from './SignIn.module.scss'

// export const SignIn: React.FC = () => {
//   const { routes } = useAppContext()
//   const { getString } = useStrings()
//   const { mutate } = useOnLogin({
//     queryParams: {
//       include_cookie: true
//     }
//   })
//   const { showError } = useToaster()
//   const onLogin = useCallback(
//     ({ username, password }: LoginForm) => {
//       mutate(
//         { login_identifier: username, password },
//         {
//           headers: { Authorization: '' }
//         }
//       )
//         .then(() => {
//           window.location.replace(window.location.origin + routes.toCODEHome())
//         })
//         .catch(error => {
//           showError(getErrorMessage(error))
//         })
//     },
//     [mutate, showError, routes]
//   )
//   const onSubmit = useCallback(
//     (data: LoginForm): void => {
//       if (data.username && data.password) {
//         onLogin(data)
//       }
//     },
//     [onLogin]
//   )

//   return (
//     <AuthLayout>
//       <Container className={css.signInContainer}>
//         <Layout.Horizontal flex={{ alignItems: 'center' }}>
//           <Text font={{ size: 'large', weight: 'bold' }} color={Color.BLACK}>
//             {getString('signIn')}
//           </Text>
//           <FlexExpander />
//           <Layout.Horizontal spacing="xsmall">
//             <Text>{getString('noAccount?')}</Text>
//             <Link to={routes.toRegister()}>{getString('signUp')}</Link>
//           </Layout.Horizontal>
//         </Layout.Horizontal>

//         <Container margin={{ top: 'xxlarge' }}>
//           <Formik<LoginForm>
//             initialValues={{ username: '', password: '' }}
//             formName="loginPageForm"
//             onSubmit={onSubmit}
//             validationSchema={Yup.object().shape({
//               username: Yup.string().required(getString('userNameRequired')),
//               password: Yup.string().required(getString('passwordRequired'))
//             })}>
//             <FormikForm>
//               <FormInput.Text name="username" label={getString('emailUser')} disabled={false} />
//               <FormInput.Text
//                 name="password"
//                 label={getString('password')}
//                 inputGroup={{ type: 'password' }}
//                 disabled={false}
//               />
//               <Button type="submit" intent="primary" loading={false} disabled={false} width="100%">
//                 {getString('signIn')}
//               </Button>
//               <BT2>HELLO WORLD</BT2>
//             </FormikForm>
//           </Formik>
//         </Container>
//         <Layout.Horizontal padding={{ top: 'medium' }} spacing="xsmall">
//           <Text>
//             <StringSubstitute
//               str={getString('bySigningIn')}
//               vars={{
//                 policy: <a href="https://harness.io/privacy"> {getString('privacyPolicy')} </a>,
//                 terms: <a href="https://harness.io/subscriptionterms"> {getString('termsOfUse')} </a>
//               }}
//             />
//           </Text>
//         </Layout.Horizontal>
//       </Container>
//     </AuthLayout>
//   )
// }

interface DataProps {
  email?: string
  password?: string
}

const signInSchema = z.object({
  email: z.string().email({ message: 'Invalid email address' }),
  password: z.string().min(6, { message: 'Password must be at least 6 characters' })
})

export default {
  title: 'Pages/Sign In',
  parameters: {
    layout: 'fullscreen'
  }
}

export function SignIn() {
  const {
    register,
    handleSubmit,
    formState: { errors }
  } = useForm({
    resolver: zodResolver(signInSchema)
  })
  const [isLoading, setIsloading] = useState<boolean>(false)

  const onSubmit = (data: DataProps) => {
    console.log(data)

    setIsloading(true)
    setTimeout(() => {
      setIsloading(false)
    }, 2000)
  }

  return (
    <div className="dark">
      <Container.Root>
        <Container.Main>
          <Container.CenteredContent>
            <Card className="card-auth bg-transparent relative z-10">
              <img
                src={bodyBlur}
                className="bg-cover bg-top opacity-[20%] max-w-[1000px] absolute -left-[calc((1000px-362px)/2)] -top-[200px] w-[1000px] h-[900px]"
              />
              <CardHeader className="card-auth-header relative z-10">
                <CardTitle className="flex flex-col place-items-center">
                  <Logo />
                  <p className="title-primary text-radial-gradient">Sign in to Gitness</p>
                </CardTitle>
              </CardHeader>
              <CardContent className="card-auth-content relative z-10">
                <form onSubmit={handleSubmit(onSubmit)} className="space-y-4 w-full flex flex-col gap-4">
                  <div className="flex flex-col gap-3">
                    <div>
                      <Label htmlFor="email" variant="sm">
                        Email
                      </Label>
                      <Input
                        id="email"
                        type="email"
                        {...register('email')}
                        placeholder="email@work.com"
                        className="form-input"
                        autoFocus
                      />
                      {errors.email && <p className="text-form-error">{errors.email.message?.toString()}</p>}
                    </div>
                    <div>
                      <Label htmlFor="password" variant="sm">
                        Password
                      </Label>
                      <Input
                        id="password"
                        type="password"
                        {...register('password')}
                        placeholder="Enter the password for your account"
                        className="form-input"
                      />
                      {errors.password && <p className="text-form-error">{errors.password.message?.toString()}</p>}
                    </div>
                  </div>
                  <Button variant="default" borderRadius="full" type="submit" loading={isLoading}>
                    {isLoading ? 'Signing in...' : 'Sign in'}
                  </Button>
                </form>
                <div className="mt-6 text-center">
                  <p className="text-sm font-light text-white/70">
                    Don&apos;t have an account? <a className="text-white">Sign up</a>
                  </p>
                </div>
              </CardContent>
            </Card>
            <FooterStrap>
              <p className="text-xs font-light text-white/40">
                By joining, you agree to <a className="text-white/60">Terms of Service</a> and{' '}
                <a className="text-white/60">Privacy Policy</a>
              </p>
            </FooterStrap>
          </Container.CenteredContent>
        </Container.Main>
      </Container.Root>
    </div>
  )
}
