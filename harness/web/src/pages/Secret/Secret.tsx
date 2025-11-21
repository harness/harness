/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
      <PageHeader title={`THIS IS A SECRET = ${secret?.identifier}`} />
    </Container>
  )
}

export default Execution
