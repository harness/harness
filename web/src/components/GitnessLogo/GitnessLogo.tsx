import React from 'react'
import { Container, Layout, Text } from '@harnessio/uicore'
import { Link } from 'react-router-dom'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import gitness from './gitness.svg'
import css from './GitnessLogo.module.scss'

export const GitnessLogo: React.FC = () => {
  const { routes } = useAppContext()
  const { getString } = useStrings()

  return (
    <Container className={css.main}>
      <Link to={routes.toCODEHome()}>
        <Layout.Horizontal spacing="small" className={css.layout}>
          <img src={gitness} width={34} height={34} />
          <Text className={css.text}>{getString('gitness')}</Text>
        </Layout.Horizontal>
      </Link>
    </Container>
  )
}
