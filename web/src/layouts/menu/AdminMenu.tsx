import React from 'react'
import { Container, Layout } from '@harness/uicore'

import { useStrings } from 'framework/strings'
import { routes } from 'RouteDefinitions'

import { NavMenu } from './NavMenu'

export const AdminMenu: React.FC = () => {
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
          label={getString('userManagement.text')}
          to={routes.toCODEUsers()}
        />
      </Layout.Vertical>
    </Container>
  )
}
