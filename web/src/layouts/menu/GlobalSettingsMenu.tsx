import { Container, Layout } from '@harness/uicore'
import React from 'react'

import { useStrings } from 'framework/strings'
import { routes } from 'RouteDefinitions'
import { NavMenu } from './NavMenu'

export const GlobalSettingsMenu: React.FC = () => {
  const { getString } = useStrings()

  return (
    <Container padding={{ top: 'medium' }}>
      <Layout.Vertical spacing="small">
        <NavMenu
          textProps={{
            iconProps: {
              size: 12
            }
          }}
          label={getString('profile')}
          to={routes.toCODEUserProfile()}
        />
      </Layout.Vertical>
    </Container>
  )
}
