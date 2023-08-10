import React from 'react'
import { Container, Icon, Layout, Text } from '@harness/uicore'
import { Link } from 'react-router-dom'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import css from './GitnessLogo.module.scss'

export const GitnessLogo: React.FC = () => {
  const { routes } = useAppContext()
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
