/*
 * Copyright 2021 Harness Inc. All rights reserved.
 * Use of this source code is governed by the PolyForm Shield 1.0.0 license
 * that can be found in the licenses directory at the root of this repository, also available at
 * https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
 */

import React, { ComponentType, LazyExoticComponent, ReactElement } from 'react'
import { useAppContext } from 'AppContext'
import type { ChildAppProps } from '@harness/microfrontends'
import ChildComponentMounter, { ChildComponentMounterProps } from './ChildComponentMounter'

export interface ChildAppMounterProps extends Omit<ChildComponentMounterProps, 'ChildComponent'> {
  ChildApp: LazyExoticComponent<ComponentType<ChildAppProps>>
}

function ChildAppMounter<T>({ ChildApp, children, ...rest }: T & ChildAppMounterProps): ReactElement {

  const { components, hooks } = useAppContext()

  return (
    <ChildComponentMounter<Pick<ChildAppProps, 'components' | 'hooks' | 'utils'>>
      ChildComponent={ChildApp as ChildComponentMounterProps['ChildComponent']}
      {...rest}
      components={components}
      hooks={hooks}
      utils={{ getLocationPathName: () => 'en' }}
    >
      {children}
    </ChildComponentMounter>
  )
}

export default ChildAppMounter
