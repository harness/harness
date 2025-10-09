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
import { Container, Layout } from '@harnessio/uicore'
import cx from 'classnames'
import { Images } from 'images'
import css from './AuthLayout.module.scss'

const AuthLayout: React.FC<React.PropsWithChildren<unknown>> = props => {
  return (
    <div className={css.layout}>
      <div className={css.imageColumn} style={{ background: `url(${Images.DarkBackground})` }}>
        <Container height={250} padding={'xxlarge'}>
          <img className={css.harnessImage} width={75} height={16} src={Images.HarnessDarkLogo} alt="" aria-hidden />
        </Container>
        <Layout.Vertical className={css.gitnessContainer}>
          <img className={cx(css.image)} src={Images.Signup} alt="" aria-hidden />
        </Layout.Vertical>
        <Container className={css.subtractContainer}>
          <img className={css.subtractImage} width={250} height={250} src={Images.GitLogo} alt="" aria-hidden />
        </Container>
      </div>
      <div className={css.cardColumn}>
        <div className={css.card}>
          <Container className={css.cardChildren}>{props.children}</Container>
        </div>
      </div>
    </div>
  )
}

export default AuthLayout
