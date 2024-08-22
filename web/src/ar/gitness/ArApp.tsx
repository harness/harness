/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react'
import { RouteComponentProps, withRouter } from 'react-router-dom'

import GitnessApp from '@ar/app/GitnessApp'
import type { CustomComponents, CustomUtils, ParentContextObj } from '@ar/MFEAppTypes'

import { handle401 } from 'AppUtils'
import { generateAlphaNumericHash } from 'utils/Utils'
import useGenerateToken from 'components/CloneCredentialDialog/useGenerateToken'
import SecretFormInput from 'pages/SecretList/components/SecretFormInput/SecretFormInput'

import { ArAppContext, ArAppProvider } from './ArAppProvider'
import getARRouteDefinitions from './utils/getARRouteDefinitions'

import './ArStyles.scss'

function ArApp(props: RouteComponentProps<Record<string, string>>) {
  const { match } = props
  const { url, params, path } = match
  const { mutate } = useGenerateToken()

  const generateToken = async () => {
    const hash = generateAlphaNumericHash(6)
    return mutate({ uid: `token_${hash}` }).then(res => {
      return res.access_token
    })
  }

  return (
    <ArAppProvider>
      <GitnessApp
        matchPath={path}
        renderUrl={url}
        on401={handle401}
        parentContextObj={
          {
            appStoreContext: ArAppContext
          } as ParentContextObj
        } // TODO: add missing items and remove typecasting
        scope={params}
        customComponents={
          {
            SecretFormInput
          } as CustomComponents
        }
        customUtils={
          {
            getRouteDefinitions: getARRouteDefinitions,
            generateToken
          } as CustomUtils
        }
      />
    </ArAppProvider>
  )
}

export default withRouter(ArApp)
