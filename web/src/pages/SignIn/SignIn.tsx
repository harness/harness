// import { useHistory } from 'react-router-dom'
import React, { useCallback, useState } from 'react'
// import { get } from 'lodash-es'
import { Button, Container, Layout, Text, TextInput } from '@harness/uicore'
import { useStrings } from 'framework/strings'
import { useOnLogin } from 'services/pm'
// import routes from 'RouteDefinitions'
// import { useAPIToken } from 'hooks/useAPIToken'

export const SignIn: React.FC = () => {
  const { getString } = useStrings()
  // const history = useHistory()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  // const [, setToken] = useAPIToken()
  const { mutate } = useOnLogin({})
  const onLogin = useCallback(() => {
    const formData = new FormData()

    formData.append('username', username)
    formData.append('password', password)

    mutate(formData as unknown as void)
      .then(_ => {
        // setToken(get(data, 'access_token'))
        // history.replace(routes.toTestPage1())
      })
      .catch(error => {
        // TODO: Use toaster to show error
        // eslint-disable-next-line no-console
        console.error({ error })
      })
  }, [mutate, username, password])

  return (
    <Layout.Vertical>
      <h1>{getString('signIn')}</h1>
      <Container>
        <Layout.Horizontal>
          <Text>Username</Text>
          <TextInput
            defaultValue={username}
            name="username"
            onChange={e => setUsername((e.target as HTMLInputElement).value)}></TextInput>
        </Layout.Horizontal>
      </Container>
      <Container>
        <Layout.Horizontal>
          <Text>{getString('password')}</Text>
          <TextInput
            defaultValue={password}
            type="password"
            name="password"
            onChange={e => setPassword((e.target as HTMLInputElement).value)}></TextInput>
        </Layout.Horizontal>
      </Container>
      <Container>
        <Button text={getString('signIn')} onClick={() => onLogin()} />
      </Container>
    </Layout.Vertical>
  )
}
