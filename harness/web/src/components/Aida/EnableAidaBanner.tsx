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

import React, { useState } from 'react'
import { Button, ButtonSize, ButtonVariation, Container, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { Link, useParams } from 'react-router-dom'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import type { Identifier } from 'utils/types'

const EnableAidaBanner: React.FC<React.PropsWithChildren<unknown>> = () => {
  const { getString } = useStrings()
  const { standalone, hooks, routingId, defaultSettingsURL } = useAppContext()
  const [aidaEnableBanner, setAidaEnableBanner] = useState<boolean>(true)
  const { orgIdentifier, projectIdentifier } = useParams<Identifier>()
  const { data: aidaSettingResponse, loading: isAidaSettingLoading } = hooks?.useGetSettingValue({
    identifier: 'aida',
    queryParams: { accountIdentifier: routingId, orgIdentifier, projectIdentifier }
  })
  return (
    <>
      {!standalone && aidaSettingResponse?.data?.value != 'true' && !isAidaSettingLoading && aidaEnableBanner && (
        <Container
          background={Color.AI_PURPLE_50}
          padding="small"
          margin={{ top: 'xsmall', right: 'small', left: 'xsmall' }}
          style={{ position: 'relative' }}>
          <Text
            font={{ variation: FontVariation.BODY2 }}
            margin={{ bottom: 'small' }}
            icon="info-messaging"
            iconProps={{ size: 15 }}>
            {getString('enableAIDAPRDescription')}
          </Text>
          <Text font={{ variation: FontVariation.BODY2_SEMI }} margin={{ bottom: 'small' }} color={Color.GREY_450}>
            {getString('enableAIDAPRMessange')}
          </Text>
          <Link to={defaultSettingsURL} color={Color.AI_PURPLE_800}>
            {getString('reviewProjectSettings')}
          </Link>
          <Button
            variation={ButtonVariation.ICON}
            minimal
            icon="main-close"
            role="close"
            iconProps={{ size: 12 }}
            style={{ position: 'absolute', top: '3px', right: '3px' }}
            size={ButtonSize.SMALL}
            onClick={() => {
              setAidaEnableBanner(false)
            }}
          />
        </Container>
      )}
    </>
  )
}

export default EnableAidaBanner
