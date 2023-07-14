import React from 'react'
import { Container, Heading } from '@harness/uicore'
import { Link } from 'react-router-dom'
import { routes } from 'RouteDefinitions'
import { useStrings } from 'framework/strings'
import css from './Spaces.module.scss'

export default function Spaces() {
  const { getString } = useStrings()

  return (
    <Container className={css.main}>
      <Heading className={css.pageHeading} level={1}>
        {getString('spaces')}
      </Heading>
      <Container className={css.pageContent}>
        <Link to={routes.toCODERepositories({ space: 'root' })}>Test Space</Link>
      </Container>
    </Container>
  )
}
