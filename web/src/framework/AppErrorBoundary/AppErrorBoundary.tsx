import React from 'react'
import { Text, Layout, Container } from '@harnessio/uicore'
import i18n from './AppErrorBoundary.i18n.json'

interface AppErrorBoundaryState {
  error?: Error
}

class AppErrorBoundary extends React.Component<unknown, AppErrorBoundaryState> {
  state: AppErrorBoundaryState = { error: undefined }

  componentDidCatch(error: Error): boolean {
    this.setState({ error })
    if (window?.bugsnagClient?.notify) {
      window?.bugsnagClient?.notify(error)
    }
    return false
  }

  render(): React.ReactNode {
    const { error } = this.state

    if (error) {
      return (
        <Layout.Vertical spacing="medium" padding="large">
          <Text>{i18n.title}</Text>
          <Text>{i18n.subtitle}</Text>
          <Text>
            {i18n.please}
            <a
              href="#"
              onClick={e => {
                e.preventDefault()
                window.location.reload()
              }}>
              {i18n.refresh}
            </a>
            {i18n.continue}
          </Text>
          {__DEV__ && (
            <React.Fragment>
              <Text font="small">{error.message}</Text>
              <Container>
                <details>
                  <summary>{i18n.stackTrace}</summary>
                  <pre>{error.stack}</pre>
                </details>
              </Container>
            </React.Fragment>
          )}
        </Layout.Vertical>
      )
    }

    return <>{this.props.children}</>
  }
}

export default AppErrorBoundary
