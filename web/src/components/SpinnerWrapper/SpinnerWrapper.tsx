import React, { CSSProperties } from 'react'
import { Layout } from '@harness/uicore'
import { Spinner } from '@blueprintjs/core'
import cx from 'classnames'
import css from './SpinnerWrapper.module.scss'

export const SpinnerWrapper = ({
  loading,
  children,
  style
}: {
  loading: boolean
  children: React.ReactNode | undefined
  style?: CSSProperties
}): JSX.Element => {
  return (
    <Layout.Vertical style={style}>
      <Layout.Horizontal className={cx(css.loadingSpinnerWrapper, { [css.hidden]: !loading })}>
        <Spinner />
      </Layout.Horizontal>
      {!loading && children}
    </Layout.Vertical>
  )
}
