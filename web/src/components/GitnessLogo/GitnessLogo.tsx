import React from 'react'
import { Container, Layout } from '@harnessio/uicore'
import { Link } from 'react-router-dom'
import { useAppContext } from 'AppContext'
import gitness from './gitness.svg'
import css from './GitnessLogo.module.scss'

export const GitnessLogo: React.FC = () => {
  const { routes } = useAppContext()

  return (
    <Container className={css.main}>
      <Link to={routes.toCODEHome()}>
        <Layout.Horizontal spacing="small" className={css.layout} padding={{ left: 'small' }}>
          <img src={gitness} width={100} height={50} />
        </Layout.Horizontal>
      </Link>
    </Container>
  )
}
