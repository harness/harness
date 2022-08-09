import React from 'react'
import cx from 'classnames'
import { Intent, IconName, Text } from '@harness/uicore'
import type { IconProps } from '@harness/uicore/dist/icons/Icon'
import css from './EvaluationStatusLabel.module.scss'

export interface EvaluationStatusProps {
  intent: Intent
  label: string
  icon?: IconName
  iconProps?: IconProps
  className?: string
}

export const EvaluationStatusLabel: React.FC<EvaluationStatusProps> = ({
  intent,
  icon,
  iconProps,
  label,
  className
}) => {
  let _icon: IconName | undefined = icon

  if (!_icon) {
    switch (intent) {
      case Intent.DANGER:
      case Intent.WARNING:
        _icon = 'warning-sign'
        break
      case Intent.SUCCESS:
        _icon = 'tick-circle'
        break
    }
  }

  return (
    <Text icon={_icon} iconProps={{ size: 9, ...iconProps }} className={cx(css.status, className, css[intent])}>
      {label}
    </Text>
  )
}
