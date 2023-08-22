import React from 'react'
import { Button, ButtonProps } from '@harnessio/uicore'
import css from './PlainButton.module.scss'

export const PlainButton: React.FC<ButtonProps> = props => <Button className={css.btn} {...props} />
