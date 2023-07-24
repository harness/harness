import React from 'react'
import { Container, Icon, Layout, Text } from '@harness/uicore'
import { Link } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { routes } from 'RouteDefinitions'
import css from './GitnessLogo.module.scss'

export const GitnessLogo: React.FC = () => {
  const { getString } = useStrings()

  return (
    <Container className={css.main}>
      <Link to={routes.toCODEHome()}>
        <Layout.Horizontal spacing="small" className={css.layout}>
          <Icon name="code" size={34} />
          <Text className={css.text}>{getString('gitness')}</Text>
        </Layout.Horizontal>
      </Link>
    </Container>
  )
}
