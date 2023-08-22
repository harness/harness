import React from 'react'
import { Container, PageHeader } from '@harnessio/uicore'
import { useParams } from 'react-router-dom'
import { useGet } from 'restful-react'
import type { CODEProps } from 'RouteDefinitions'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import type { TypesSecret } from 'services/code'
import css from './Secret.module.scss'

const Execution = () => {
  const space = useGetSpaceParam()
  const { secret: secretName } = useParams<CODEProps>()

  const {
    data: secret
    // error,
    // loading,
    // refetch
    // response
  } = useGet<TypesSecret>({
    path: `/api/v1/secrets/${space}/+/${secretName}`
  })

  return (
    <Container className={css.main}>
      <PageHeader title={`THIS IS A SECRET = ${secret?.uid}`} />
    </Container>
  )
}

export default Execution
