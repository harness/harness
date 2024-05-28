/*
 * Copyright 2022 Harness Inc. All rights reserved.
 * Use of this source code is governed by the PolyForm Shield 1.0.0 license
 * that can be found in the licenses directory at the root of this repository, also available at
 * https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
 */

import React from 'react'
import { PageBody, PageError } from '@harnessio/uicore'

const devErrorMsg = 'This app is rendered as a microfrontend. It looks like it is not reachable.'

export default function ChildAppError(): React.ReactElement {
  return (
    <PageBody>
      <PageError message={__DEV__ ? devErrorMsg : ''} />
    </PageBody>
  )
}
