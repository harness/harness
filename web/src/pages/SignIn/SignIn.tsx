import { useHistory } from 'react-router-dom'
import React, { useCallback, useState } from 'react'
import { Button, Container, Layout, Text, TextInput } from '@harness/uicore'
import { useStrings } from 'framework/strings'
import { routes } from 'RouteDefinitions'
import { useOnLogin } from 'services/code'
import { useAPIToken } from 'hooks/useAPIToken'

export const SignIn: React.FC = () => {
  const { getString } = useStrings()
  const [, setToken] = useAPIToken()
  const history = useHistory()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const { mutate } = useOnLogin({})
  const onLogin = useCallback(() => {
    const formData = new FormData()

    formData.append('username', username)
    formData.append('password', password)

    mutate(formData as unknown as void, { headers: { Authorization: '' } })
      .then(response => {
        setToken(response.access_token as string)
        history.replace(routes.toCODESpaces())
      })
      .catch(error => {
        // TODO: Show error message as a toast
        console.error({ error }) // eslint-disable-line no-console
      })
  }, [mutate, username, password, setToken, history])

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
        <Button text={getString('signIn')} onClick={onLogin} />
      </Container>
    </Layout.Vertical>
  )
}
