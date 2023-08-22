import React from 'react'
import { Container, Layout } from '@harnessio/uicore'
import cx from 'classnames'
import { Images } from 'images'
import css from './AuthLayout.module.scss'

const AuthLayout: React.FC<React.PropsWithChildren<unknown>> = props => {
  return (
    <div className={css.layout}>
      <div className={css.imageColumn} style={{ background: `url(${Images.DarkBackground})` }}>
        <Container className={css.subtractContainer}>
          <img className={css.subtractImage} width={250} height={250} src={Images.Subtract} alt="" aria-hidden />
        </Container>
        <Layout.Vertical className={css.imageContainer}>
          <img className={cx(css.image)} src={Images.Signup} alt="" aria-hidden />
        </Layout.Vertical>
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
