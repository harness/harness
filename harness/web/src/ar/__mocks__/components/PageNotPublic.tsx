/*
 * Copyright 2023 Harness Inc. All rights reserved.
 * Use of this source code is governed by the PolyForm Shield 1.0.0 license
 * that can be found in the licenses directory at the root of this repository, also available at
 * https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
 */

import React from 'react'

import { Text, Layout, Container } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useStrings } from '@ar/frameworks/strings'

const PageNotPublic: React.FC = () => {
  const { getString } = useStrings()
  return (
    <Container height="100vh" width="100%" flex={{ justifyContent: 'center', alignItems: 'center' }}>
      <Layout.Vertical spacing="large">
        <Text color={Color.GREY_1000} font={{ variation: FontVariation.H1 }}>
          {getString('publicAccess.oopsPageNotPublic')}
        </Text>
        <Text color={Color.GREY_1000} font={{ variation: FontVariation.H5 }} margin={{ bottom: 'xlarge' }}>
          {getString('publicAccess.tryOtherOptions')}
        </Text>
      </Layout.Vertical>
    </Container>
  )
}

export default PageNotPublic
