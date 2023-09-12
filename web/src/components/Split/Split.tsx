import React from 'react'
import cx from 'classnames'
import SplitPane, { type SplitPaneProps } from 'react-split-pane'
import css from './Split.module.scss'

export const Split: React.FC<SplitPaneProps> = ({ className, ...props }) => (
  <SplitPane className={cx(css.main, className)} {...props} />
)
