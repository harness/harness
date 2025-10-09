/*
 * Copyright 2022 Harness Inc. All rights reserved.
 * Use of this source code is governed by the PolyForm Shield 1.0.0 license
 * that can be found in the licenses directory at the root of this repository, also available at
 * https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
 */

import React, { ReactElement } from 'react'
import { Text, Popover, TextProps, Layout } from '@harnessio/uicore'
import type { IconName } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { IPopoverProps, PopoverInteractionKind, Position } from '@blueprintjs/core'
import ReactTimeago, { ReactTimeagoProps } from 'react-timeago'
import moment from 'moment'
import { useStrings } from 'framework/strings'
import css from './TimePopoverWithLocal.module.scss'

type ReactTimeagoPropsWithoutDate = Omit<ReactTimeagoProps, 'date'>
type CommonTextProps = Omit<TextProps, 'title'>
type CommonReactTimeagoProps = Omit<ReactTimeagoPropsWithoutDate, 'title'>

interface TimePopoverProps extends CommonTextProps, CommonReactTimeagoProps {
  date?: ReactTimeagoProps['date']
  popoverProps?: IPopoverProps
  icon?: IconName
  className?: string
  title?: string | undefined
  time: number
}

export const DATE_TIME_PARSE_FORMAT = 'MMM DD, YYYY hh:mm:ss A'
export const DATE_PARSE_FORMAT = 'MMM DD, YYYY'
export const TIME_PARSE_FORMAT = 'hh:mm:ss A'

enum TimeZone {
  UTC = 'UTC',
  LOCAL = 'LOCAL'
}

export function DateTimeWithLocalContentInline({ time }: { time: number }): JSX.Element {
  const { getString } = useStrings()
  return (
    <Layout.Vertical margin={{ right: '4px' }}>
      <Layout.Horizontal className={css.timeWrapper}>
        <Text color={Color.GREY_600} className={css.time}>
          {moment(time).format(DATE_PARSE_FORMAT)}
        </Text>
        <Text color={Color.GREY_450}>{getString('at')}</Text>
        <Text color={Color.GREY_600} className={css.time}>
          {moment(time).format(TIME_PARSE_FORMAT)}
        </Text>
      </Layout.Horizontal>
    </Layout.Vertical>
  )
}

export function DateTimeContent({ time }: { time: number }): JSX.Element {
  return (
    <Layout.Vertical>
      <Layout.Horizontal spacing={'small'} className={css.timeWrapper}>
        <Text
          color={Color.PRIMARY_1}
          font={{ variation: FontVariation.SMALL_BOLD }}
          margin={0}
          className={css.timezone}>
          {TimeZone.UTC}
        </Text>
        <Text color={Color.PRIMARY_1} className={css.time} font={{ variation: FontVariation.SMALL_BOLD }}>
          {moment(time).utc().format(DATE_PARSE_FORMAT)}
        </Text>
      </Layout.Horizontal>

      <Layout.Horizontal spacing={'small'} className={css.timeWrapper}>
        <Text
          color={Color.PRIMARY_1}
          font={{ variation: FontVariation.SMALL_BOLD }}
          margin={0}
          className={css.timezone}>
          {TimeZone.LOCAL}
        </Text>
        <Text color={Color.PRIMARY_1} className={css.time} font={{ variation: FontVariation.SMALL_BOLD }}>
          {moment(time).format(DATE_PARSE_FORMAT)}
        </Text>
      </Layout.Horizontal>
    </Layout.Vertical>
  )
}

export function TimePopoverWithLocal(props: TimePopoverProps): ReactElement {
  const { time, popoverProps, icon, className, ...textProps } = props
  return (
    <Popover
      interactionKind={PopoverInteractionKind.HOVER}
      position={Position.TOP}
      //   className={Classes.DARK}
      {...popoverProps}>
      <Text inline {...textProps} icon={icon} className={className}>
        <ReactTimeago date={time} live title={''} className={css.noWrap} />
      </Text>
      <Layout.Vertical padding="medium">
        <DateTimeWithLocalContentInline time={time} />
      </Layout.Vertical>
    </Popover>
  )
}
