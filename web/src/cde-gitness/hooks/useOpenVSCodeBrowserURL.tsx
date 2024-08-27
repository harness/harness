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

import { useEffect, useState } from 'react'
import { OpenapiGetTokenResponse, useGetToken } from 'services/cde'

export const useOpenVSCodeBrowserURL = () => {
  const { data: tokenData, refetch: refetchToken } = useGetToken({
    accountIdentifier: '',
    projectIdentifier: '',
    orgIdentifier: '',
    gitspace_identifier: '',
    lazy: true
  })

  const [temporaryToken, setTemporaryToken] = useState<OpenapiGetTokenResponse | undefined>({
    gitspace_token: undefined
  })
  const [selectedRowUrl, setSelectedRowUrl] = useState<string | undefined>('')

  useEffect(() => {
    if (temporaryToken?.gitspace_token) {
      window.open(`${selectedRowUrl}&token=${temporaryToken?.gitspace_token}`, '_blank')
    }
  }, [temporaryToken, selectedRowUrl])

  useEffect(() => {
    if (tokenData?.gitspace_token !== temporaryToken?.gitspace_token && tokenData) {
      setTemporaryToken(tokenData)
    }
  }, [temporaryToken, tokenData])

  return { refetchToken, setSelectedRowUrl }
}
